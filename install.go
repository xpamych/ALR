// This file was originally part of the project "LURE - Linux User REpository", created by Elara Musayelyan.
// It has been modified as part of "ALR - Any Linux Repository" by the ALR Authors.
//
// ALR - Any Linux Repository
// Copyright (C) 2025 The ALR Authors
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
	"strings"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/build"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	appbuilder "gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils/app_builder"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/manager"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/utils"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/types"
)

func InstallCmd() *cli.Command {
	return &cli.Command{
		Name:    "install",
		Usage:   gotext.Get("Install a new package"),
		Aliases: []string{"in"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "clean",
				Aliases: []string{"c"},
				Usage:   gotext.Get("Build package from scratch even if there's an already built package available"),
			},
		},
		Action: utils.RootNeededAction(func(c *cli.Context) error {
			args := c.Args()
			if args.Len() < 1 {
				return cliutils.FormatCliExit(gotext.Get("Command install expected at least 1 argument, got %d", args.Len()), nil)
			}


			installer, installerClose, err := build.GetSafeInstaller()
			if err != nil {
				return err
			}
			defer installerClose()


			scripter, scripterClose, err := build.GetSafeScriptExecutor()
			if err != nil {
				return err
			}
			defer scripterClose()

			ctx := c.Context

			deps, err := appbuilder.
				New(ctx).
				WithConfig().
				WithDB().
				WithRepos().
				WithDistroInfo().
				WithManager().
				Build()
			if err != nil {
				return err
			}
			defer deps.Defer()

			builder, err := build.NewMainBuilder(
				deps.Cfg,
				deps.Manager,
				deps.Repos,
				scripter,
				installer,
			)
			if err != nil {
				return err
			}

			_, err = builder.InstallPkgs(
				ctx,
				&build.BuildArgs{
					Opts: &types.BuildOpts{
						Clean:       c.Bool("clean"),
						Interactive: c.Bool("interactive"),
					},
					Info:       deps.Info,
					PkgFormat_: build.GetPkgFormat(deps.Manager),
				},
				args.Slice(),
			)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error when installing the package"), err)
			}

			return nil
		}),
		BashComplete: cliutils.BashCompleteWithError(func(c *cli.Context) error {
			ctx := c.Context
			deps, err := appbuilder.
				New(ctx).
				WithConfig().
				WithDB().
				WithManager().
				Build()
			if err != nil {
				return err
			}
			defer deps.Defer()

			seen := make(map[string]struct{})

			var prefix string
			if c.Args().Len() > 0 {
				prefix = c.Args().Get(c.Args().Len() - 1)
				if strings.HasPrefix(prefix, "-") {
					prefix = ""
				}
			}

			result, err := deps.DB.GetPkgs(c.Context, "true")
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error getting packages"), err)
			}

			for _, pkg := range result {
				if prefix == "" || strings.HasPrefix(pkg.Name, prefix) {
					if _, ok := seen[pkg.Name]; !ok {
						seen[pkg.Name] = struct{}{}
						fmt.Println(pkg.Name)
					}
				}
			}

			sysPkgs, err := deps.Manager.ListAvailable(prefix)
			if err != nil {
				slog.Debug("failed to list system packages", "err", err)
			} else {
				for _, name := range sysPkgs {
					if _, ok := seen[name]; !ok {
						seen[name] = struct{}{}
						fmt.Println(name)
					}
				}
			}

			return nil
		}),
	}
}

func RemoveCmd() *cli.Command {
	return &cli.Command{
		Name:    "remove",
		Usage:   gotext.Get("Remove an installed package"),
		Aliases: []string{"rm"},
		BashComplete: cliutils.BashCompleteWithError(func(c *cli.Context) error {
			ctx := c.Context

			deps, err := appbuilder.
				New(ctx).
				WithConfig().
				WithDB().
				WithManager().
				Build()
			if err != nil {
				return cli.Exit(err, 1)
			}
			defer deps.Defer()

			installedAlrPackages := map[string]string{}
			installed, err := deps.Manager.ListInstalled(&manager.Opts{})
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error listing installed packages"), err)
			}
			for pkgName, version := range installed {
				matches := build.RegexpALRPackageName.FindStringSubmatch(pkgName)
				if matches != nil {
					packageName := matches[build.RegexpALRPackageName.SubexpIndex("package")]
					repoName := matches[build.RegexpALRPackageName.SubexpIndex("repo")]
					installedAlrPackages[fmt.Sprintf("%s/%s", repoName, packageName)] = version
				}
			}

			result, err := deps.DB.GetPkgs(c.Context, "true")
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error getting packages"), err)
			}

			for _, pkg := range result {
				_, ok := installedAlrPackages[fmt.Sprintf("%s/%s", pkg.Repository, pkg.Name)]
				if !ok {
					continue
				}
				fmt.Println(pkg.Name)
			}

			return nil
		}),
		Action: utils.RootNeededAction(func(c *cli.Context) error {
			args := c.Args()
			if args.Len() < 1 {
				return cliutils.FormatCliExit(gotext.Get("Command remove expected at least 1 argument, got %d", args.Len()), nil)
			}

			deps, err := appbuilder.
				New(c.Context).
				WithManager().
				Build()
			if err != nil {
				return err
			}
			defer deps.Defer()

			if err := deps.Manager.Remove(&manager.Opts{
				NoConfirm: !c.Bool("interactive"),
			}, c.Args().Slice()...); err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error removing packages"), err)
			}

			return nil
		}),
	}
}
