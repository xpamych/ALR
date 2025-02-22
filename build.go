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
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	database "gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/osutils"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/build"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/distro"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/manager"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/repos"
)

func BuildCmd() *cli.Command {
	return &cli.Command{
		Name:  "build",
		Usage: gotext.Get("Build a local package"),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "script",
				Aliases: []string{"s"},
				Value:   "alr.sh",
				Usage:   gotext.Get("Path to the build script"),
			},
			&cli.StringFlag{
				Name:    "script-package",
				Aliases: []string{"sp"},
				Usage:   gotext.Get("Specify package in script (for multi package script only)"),
			},
			&cli.StringFlag{
				Name:    "package",
				Aliases: []string{"p"},
				Usage:   gotext.Get("Name of the package to build and its repo (example: default/go-bin)"),
			},
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
				slog.Error(gotext.Get("Error initialization database"), "err", err)
				os.Exit(1)
			}

			var script string
			var packages []string

			// Проверяем, установлен ли флаг script (-s)

			repoDir := cfg.GetPaths(ctx).RepoDir

			switch {
			case c.IsSet("script"):
				script = c.String("script")
				packages = append(packages, c.String("script-package"))
			case c.IsSet("package"):
				// TODO: handle multiple packages
				packageInput := c.String("package")

				arr := strings.Split(packageInput, "/")
				var packageSearch string
				if len(arr) == 2 {
					packageSearch = arr[1]
				} else {
					packageSearch = arr[0]
				}

				pkgs, _, _ := rs.FindPkgs(ctx, []string{packageSearch})
				pkg, ok := pkgs[packageSearch]
				if len(pkg) < 1 || !ok {
					slog.Error(gotext.Get("Package not found"))
					os.Exit(1)
				}

				if pkg[0].BasePkgName != "" {
					script = filepath.Join(repoDir, pkg[0].Repository, pkg[0].BasePkgName, "alr.sh")
					packages = append(packages, pkg[0].Name)
				} else {
					script = filepath.Join(repoDir, pkg[0].Repository, pkg[0].Name, "alr.sh")
				}
			default:
				script = filepath.Join(repoDir, "alr.sh")
			}

			// Проверка автоматического пулла репозиториев
			if cfg.AutoPull(ctx) {
				err := rs.Pull(ctx, cfg.Repos(ctx))
				if err != nil {
					slog.Error(gotext.Get("Error pulling repositories"), "err", err)
					os.Exit(1)
				}
			}

			// Обнаружение менеджера пакетов
			mgr := manager.Detect()
			if mgr == nil {
				slog.Error(gotext.Get("Unable to detect a supported package manager on the system"))
				os.Exit(1)
			}

			info, err := distro.ParseOSRelease(ctx)
			if err != nil {
				slog.Error(gotext.Get("Error parsing os release"), "err", err)
				os.Exit(1)
			}

			builder := build.NewBuilder(
				ctx,
				types.BuildOpts{
					Packages:    packages,
					Script:      script,
					Manager:     mgr,
					Clean:       c.Bool("clean"),
					Interactive: c.Bool("interactive"),
				},
				rs,
				info,
				cfg,
			)

			// Сборка пакета
			pkgPaths, _, err := builder.BuildPackage(ctx)
			if err != nil {
				slog.Error(gotext.Get("Error building package"), "err", err)
				os.Exit(1)
			}

			// Получение текущей рабочей директории
			wd, err := os.Getwd()
			if err != nil {
				slog.Error(gotext.Get("Error getting working directory"), "err", err)
				os.Exit(1)
			}

			// Перемещение собранных пакетов в рабочую директорию
			for _, pkgPath := range pkgPaths {
				name := filepath.Base(pkgPath)
				err = osutils.Move(pkgPath, filepath.Join(wd, name))
				if err != nil {
					slog.Error(gotext.Get("Error moving the package"), "err", err)
					os.Exit(1)
				}
			}

			return nil
		},
	}
}
