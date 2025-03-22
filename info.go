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

	"github.com/jeandeaual/go-locale"
	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	database "gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/overrides"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/distro"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/repos"
)

func InfoCmd() *cli.Command {
	return &cli.Command{
		Name:  "info",
		Usage: gotext.Get("Print information about a package"),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   gotext.Get("Show all information, not just for the current distro"),
			},
		},
		BashComplete: func(c *cli.Context) {
			ctx := c.Context
			cfg := config.New()
			err := cfg.Load()
			if err != nil {
				slog.Error(gotext.Get("Error loading config"), "err", err)
				os.Exit(1)
			}

			db := database.New(cfg)
			err = db.Init(ctx)
			if err != nil {
				slog.Error(gotext.Get("Error initialization database"), "err", err)
				os.Exit(1)
			}

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
		Action: func(c *cli.Context) error {
			ctx := c.Context

			cfg := config.New()
			err := cfg.Load()
			if err != nil {
				slog.Error(gotext.Get("Error loading config"), "err", err)
				os.Exit(1)
			}

			db := database.New(cfg)
			err = db.Init(ctx)
			if err != nil {
				slog.Error(gotext.Get("Error initialization database"), "err", err)
				os.Exit(1)
			}
			rs := repos.New(cfg, db)

			args := c.Args()
			if args.Len() < 1 {
				slog.Error(gotext.Get("Command info expected at least 1 argument, got %d", args.Len()))
				os.Exit(1)
			}

			if cfg.AutoPull() {
				err := rs.Pull(ctx, cfg.Repos())
				if err != nil {
					slog.Error(gotext.Get("Error pulling repos"), "err", err)
					os.Exit(1)
				}
			}

			found, _, err := rs.FindPkgs(ctx, args.Slice())
			if err != nil {
				slog.Error(gotext.Get("Error finding packages"), "err", err)
				os.Exit(1)
			}

			if len(found) == 0 {
				os.Exit(1)
			}

			pkgs := cliutils.FlattenPkgs(ctx, found, "show", c.Bool("interactive"))

			var names []string
			all := c.Bool("all")

			systemLang, err := locale.GetLanguage()
			if err != nil {
				slog.Error("Can't detect system language", "err", err)
				os.Exit(1)
			}
			if systemLang == "" {
				systemLang = "en"
			}

			if !all {
				info, err := distro.ParseOSRelease(ctx)
				if err != nil {
					slog.Error(gotext.Get("Error parsing os-release file"), "err", err)
					os.Exit(1)
				}
				names, err = overrides.Resolve(
					info,
					overrides.DefaultOpts.
						WithLanguages([]string{systemLang}),
				)
				if err != nil {
					slog.Error(gotext.Get("Error resolving overrides"), "err", err)
					os.Exit(1)
				}
			}

			for _, pkg := range pkgs {
				if !all {
					err = yaml.NewEncoder(os.Stdout).Encode(overrides.ResolvePackage(&pkg, names))
					if err != nil {
						slog.Error(gotext.Get("Error encoding script variables"), "err", err)
						os.Exit(1)
					}
				} else {
					err = yaml.NewEncoder(os.Stdout).Encode(pkg)
					if err != nil {
						slog.Error(gotext.Get("Error encoding script variables"), "err", err)
						os.Exit(1)
					}
				}

				fmt.Println("---")
			}

			return nil
		},
	}
}
