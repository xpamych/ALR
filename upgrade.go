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
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"
	"go.elara.ws/vercmp"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	database "gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/overrides"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/build"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/distro"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/manager"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/repos"
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
			ctx := c.Context

			cfg := config.New()
			db := database.New(cfg)
			rs := repos.New(cfg, db)
			err := db.Init(ctx)
			if err != nil {
				slog.Error(gotext.Get("Error db init"), "err", err)
				os.Exit(1)
			}

			info, err := distro.ParseOSRelease(ctx)
			if err != nil {
				slog.Error(gotext.Get("Error parsing os-release file"), "err", err)
				os.Exit(1)
			}

			mgr := manager.Detect()
			if mgr == nil {
				slog.Error(gotext.Get("Unable to detect a supported package manager on the system"))
				os.Exit(1)
			}

			if cfg.AutoPull(ctx) {
				err = rs.Pull(ctx, cfg.Repos(ctx))
				if err != nil {
					slog.Error(gotext.Get("Error pulling repos"), "err", err)
					os.Exit(1)
				}
			}

			updates, err := checkForUpdates(ctx, mgr, cfg, rs, info)
			if err != nil {
				slog.Error(gotext.Get("Error checking for updates"), "err", err)
				os.Exit(1)
			}

			if len(updates) > 0 {
				builder := build.NewBuilder(
					ctx,
					types.BuildOpts{
						Manager:     mgr,
						Clean:       c.Bool("clean"),
						Interactive: c.Bool("interactive"),
					},
					rs,
					info,
					cfg,
				)
				builder.InstallPkgs(ctx, updates, nil, types.BuildOpts{
					Manager:     mgr,
					Clean:       c.Bool("clean"),
					Interactive: c.Bool("interactive"),
				})
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
	cfg *config.ALRConfig,
	rs *repos.Repos,
	info *distro.OSRelease,
) ([]database.Package, error) {
	installed, err := mgr.ListInstalled(nil)
	if err != nil {
		return nil, err
	}

	pkgNames := maps.Keys(installed)
	found, _, err := rs.FindPkgs(ctx, pkgNames)
	if err != nil {
		return nil, err
	}

	var out []database.Package
	for pkgName, pkgs := range found {
		if slices.Contains(cfg.IgnorePkgUpdates(ctx), pkgName) {
			continue
		}

		if len(pkgs) > 1 {
			// Puts the element with the highest version first
			slices.SortFunc(pkgs, func(a, b database.Package) int {
				return vercmp.Compare(a.Version, b.Version)
			})
		}

		// First element is the package we want to install
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
	return out, nil
}
