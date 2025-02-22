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
	"os"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	database "gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/build"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/distro"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/manager"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/repos"
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
				Usage:   "Build package from scratch even if there's an already built package available",
			},
		},
		Action: func(c *cli.Context) error {
			ctx := c.Context

			args := c.Args()
			if args.Len() < 1 {
				slog.Error(gotext.Get("Command install expected at least 1 argument, got %d", args.Len()))
				os.Exit(1)
			}

			mgr := manager.Detect()
			if mgr == nil {
				slog.Error(gotext.Get("Unable to detect a supported package manager on the system"))
				os.Exit(1)
			}

			cfg := config.New()
			db := database.New(cfg)
			rs := repos.New(cfg, db)
			err := db.Init(ctx)
			if err != nil {
				slog.Error(gotext.Get("Error initialization database"), "err", err)
				os.Exit(1)
			}

			if cfg.AutoPull(ctx) {
				err := rs.Pull(ctx, cfg.Repos(ctx))
				if err != nil {
					slog.Error(gotext.Get("Error pulling repositories"), "err", err)
					os.Exit(1)
				}
			}

			found, notFound, err := rs.FindPkgs(ctx, args.Slice())
			if err != nil {
				slog.Error(gotext.Get("Error finding packages"), "err", err)
				os.Exit(1)
			}

			pkgs := cliutils.FlattenPkgs(ctx, found, "install", c.Bool("interactive"))

			opts := types.BuildOpts{
				Manager:     mgr,
				Clean:       c.Bool("clean"),
				Interactive: c.Bool("interactive"),
			}

			info, err := distro.ParseOSRelease(ctx)
			if err != nil {
				slog.Error(gotext.Get("Error parsing os release"), "err", err)
				os.Exit(1)
			}

			builder := build.NewBuilder(
				ctx,
				opts,
				rs,
				info,
				cfg,
			)

			builder.InstallPkgs(ctx, pkgs, notFound, types.BuildOpts{
				Manager:     mgr,
				Clean:       c.Bool("clean"),
				Interactive: c.Bool("interactive"),
			})
			return nil
		},
		BashComplete: func(c *cli.Context) {
			cfg := config.New()
			db := database.New(cfg)
			result, err := db.GetPkgs(c.Context, "true")
			if err != nil {
				slog.Error(gotext.Get("Error getting packages"), "err", err)
				os.Exit(1)
			}
			defer result.Close()

			for result.Next() {
				var pkg database.Package
				err = result.StructScan(&pkg)
				if err != nil {
					slog.Error(gotext.Get("Error iterating over packages"), "err", err)
					os.Exit(1)
				}

				fmt.Println(pkg.Name)
			}
		},
	}
}

func RemoveCmd() *cli.Command {
	return &cli.Command{
		Name:    "remove",
		Usage:   gotext.Get("Remove an installed package"),
		Aliases: []string{"rm"},
		Action: func(c *cli.Context) error {
			args := c.Args()
			if args.Len() < 1 {
				slog.Error(gotext.Get("Command remove expected at least 1 argument, got %d", args.Len()))
				os.Exit(1)
			}

			mgr := manager.Detect()
			if mgr == nil {
				slog.Error(gotext.Get("Unable to detect a supported package manager on the system"))
				os.Exit(1)
			}

			err := mgr.Remove(nil, c.Args().Slice()...)
			if err != nil {
				slog.Error(gotext.Get("Error removing packages"), "err", err)
				os.Exit(1)
			}

			return nil
		},
	}
}
