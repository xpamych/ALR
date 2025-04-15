// This file was originally part of the project "LURE - Linux User REpository", created by Elara Musayelyan.
// It has been modified as part of "ALR - Any Linux Repository" by Евгений Храмов.
//
// ALR - Any Linux Repository
// Copyright (C) 2025 Евгений Храмов
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"log/slog"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	appbuilder "gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils/app_builder"
	database "gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/utils"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/build"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/manager"
)

func ListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Usage:   gotext.Get("List ALR repo packages"),
		Aliases: []string{"ls"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "installed",
				Aliases: []string{"I"},
			},
		},
		Action: func(c *cli.Context) error {
			if err := utils.ExitIfCantDropCapsToAlrUserNoPrivs(); err != nil {
				return err
			}

			ctx := c.Context

			deps, err := appbuilder.
				New(ctx).
				WithConfig().
				WithDB().
				// autoPull only
				WithRepos().
				Build()
			if err != nil {
				return err
			}
			defer deps.Defer()

			cfg := deps.Cfg
			db := deps.DB

			where := "true"
			args := []any(nil)
			if c.NArg() > 0 {
				where = "name LIKE ? OR json_array_contains(provides, ?)"
				args = []any{c.Args().First(), c.Args().First()}
			}

			result, err := db.GetPkgs(ctx, where, args...)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error getting packages"), err)
			}
			defer result.Close()

			installedAlrPackages := map[string]string{}
			if c.Bool("installed") {
				mgr := manager.Detect()
				if mgr == nil {
					return cli.Exit(gotext.Get("Unable to detect a supported package manager on the system"), 1)
				}

				installed, err := mgr.ListInstalled(&manager.Opts{})
				if err != nil {
					slog.Error(gotext.Get("Error listing installed packages"), "err", err)
					return cli.Exit(err, 1)
				}

				for pkgName, version := range installed {
					matches := build.RegexpALRPackageName.FindStringSubmatch(pkgName)
					if matches != nil {
						packageName := matches[build.RegexpALRPackageName.SubexpIndex("package")]
						repoName := matches[build.RegexpALRPackageName.SubexpIndex("repo")]
						installedAlrPackages[fmt.Sprintf("%s/%s", repoName, packageName)] = version
					}
				}
			}

			for result.Next() {
				var pkg database.Package
				err := result.StructScan(&pkg)
				if err != nil {
					return cli.Exit(err, 1)
				}

				if slices.Contains(cfg.IgnorePkgUpdates(), pkg.Name) {
					continue
				}

				version := pkg.Version
				if c.Bool("installed") {
					instVersion, ok := installedAlrPackages[fmt.Sprintf("%s/%s", pkg.Repository, pkg.Name)]
					if !ok {
						continue
					} else {
						version = instVersion
					}
				}

				fmt.Printf("%s/%s %s\n", pkg.Repository, pkg.Name, version)
			}

			return nil
		},
	}
}
