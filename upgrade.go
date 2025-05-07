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
	"context"
	"fmt"
	"log/slog"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"
	"go.elara.ws/vercmp"
	"golang.org/x/exp/maps"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	appbuilder "gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils/app_builder"
	database "gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/overrides"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/utils"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/build"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/distro"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/manager"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/search"
)

func UpgradeCmd() *cli.Command {
	return &cli.Command{
		Name:    "upgrade",
		Usage:   gotext.Get("Upgrade all installed packages"),
		Aliases: []string{"up"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "clean",
				Aliases: []string{"c"},
				Usage:   gotext.Get("Build package from scratch even if there's an already built package available"),
			},
		},
		Action: func(c *cli.Context) error {
			if err := utils.ExitIfNotRoot(); err != nil {
				return err
			}

			if err := utils.ExitIfCantDropCapsToAlrUser(); err != nil {
				return err
			}

			installer, installerClose, err := build.GetSafeInstaller()
			if err != nil {
				return err
			}
			defer installerClose()

			if err := utils.ExitIfCantSetNoNewPrivs(); err != nil {
				return err
			}

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

			updates, err := checkForUpdates(ctx, deps.Manager, deps.DB, deps.Info)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error checking for updates"), err)
			}

			if len(updates) > 0 {
				err = builder.InstallALRPackages(
					ctx,
					&build.BuildArgs{
						Opts: &types.BuildOpts{
							Clean:       c.Bool("clean"),
							Interactive: c.Bool("interactive"),
						},
						Info:       deps.Info,
						PkgFormat_: build.GetPkgFormat(deps.Manager),
					},
					updates,
				)
				if err != nil {
					return cliutils.FormatCliExit(gotext.Get("Error checking for updates"), err)
				}
			} else {
				slog.Info(gotext.Get("There is nothing to do."))
			}

			return nil
		},
	}
}

func checkForUpdates(
	ctx context.Context,
	mgr manager.Manager,
	db *database.Database,
	info *distro.OSRelease,
) ([]database.Package, error) {
	installed, err := mgr.ListInstalled(nil)
	if err != nil {
		return nil, err
	}

	pkgNames := maps.Keys(installed)

	s := search.New(db)

	var out []database.Package
	for _, pkgName := range pkgNames {
		matches := build.RegexpALRPackageName.FindStringSubmatch(pkgName)
		if matches != nil {
			packageName := matches[build.RegexpALRPackageName.SubexpIndex("package")]
			repoName := matches[build.RegexpALRPackageName.SubexpIndex("repo")]

			pkgs, err := s.Search(
				ctx,
				search.NewSearchOptions().
					WithName(packageName).
					WithRepository(repoName).
					Build(),
			)
			if err != nil {
				return nil, err
			}

			if len(pkgs) == 0 {
				continue
			}

			pkg := pkgs[0]

			repoVer := pkg.Version
			releaseStr := overrides.ReleasePlatformSpecific(pkg.Release, info)

			if pkg.Release != 0 && pkg.Epoch == 0 {
				repoVer = fmt.Sprintf("%s-%s", pkg.Version, releaseStr)
			} else if pkg.Release != 0 && pkg.Epoch != 0 {
				repoVer = fmt.Sprintf("%d:%s-%s", pkg.Epoch, pkg.Version, releaseStr)
			}

			c := vercmp.Compare(repoVer, installed[pkgName])
			if c == 0 || c == -1 {
				continue
			} else if c == 1 {
				out = append(out, pkg)
			}
		}

	}

	return out, nil
}
