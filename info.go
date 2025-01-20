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
	"os"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/overrides"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/distro"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/loggerctx"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/repos"
)

var infoCmd = &cli.Command{
	Name:  "info",
	Usage: "Print information about a package",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Show all information, not just for the current distro",
		},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		log := loggerctx.From(ctx)

		args := c.Args()
		if args.Len() < 1 {
			log.Fatalf("Command info expected at least 1 argument, got %d", args.Len()).Send()
		}

		err := repos.Pull(ctx, config.Config(ctx).Repos)
		if err != nil {
			log.Fatal("Error pulling repositories").Err(err).Send()
		}

		found, _, err := repos.FindPkgs(ctx, args.Slice())
		if err != nil {
			log.Fatal("Error finding packages").Err(err).Send()
		}

		if len(found) == 0 {
			os.Exit(1)
		}

		pkgs := cliutils.FlattenPkgs(ctx, found, "show", c.Bool("interactive"))

		var names []string
		all := c.Bool("all")

		if !all {
			info, err := distro.ParseOSRelease(ctx)
			if err != nil {
				log.Fatal("Error parsing os-release file").Err(err).Send()
			}
			names, err = overrides.Resolve(
				info,
				overrides.DefaultOpts.
					WithLanguages([]string{config.SystemLang()}),
			)
			if err != nil {
				log.Fatal("Error resolving overrides").Err(err).Send()
			}
		}

		for _, pkg := range pkgs {
			if !all {
				err = yaml.NewEncoder(os.Stdout).Encode(overrides.ResolvePackage(&pkg, names))
				if err != nil {
					log.Fatal("Error encoding script variables").Err(err).Send()
				}
			} else {
				err = yaml.NewEncoder(os.Stdout).Encode(pkg)
				if err != nil {
					log.Fatal("Error encoding script variables").Err(err).Send()
				}
			}

			fmt.Println("---")
		}

		return nil
	},
}
