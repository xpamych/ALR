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
	"os"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/gen"
)

func GenCmd() *cli.Command {
	return &cli.Command{
		Name:    "generate",
		Usage:   gotext.Get("Generate a ALR script from a template"),
		Aliases: []string{"gen"},
		Subcommands: []*cli.Command{
			{
				Name:  "pip",
				Usage: gotext.Get("Generate a ALR script for a pip module"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Required: true,
					},
					&cli.StringFlag{
						Name:     "version",
						Aliases:  []string{"v"},
						Required: true,
					},
					&cli.StringFlag{
						Name:    "description",
						Aliases: []string{"d"},
					},
				},
				Action: func(c *cli.Context) error {
					return gen.Pip(os.Stdout, gen.PipOptions{
						Name:        c.String("name"),
						Version:     c.String("version"),
						Description: c.String("description"),
					})
				},
			},
			{
				Name:  "aur",
				Usage: gotext.Get("Generate a ALR script for an AUR package"),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Aliases:  []string{"n"},
						Required: true,
						Usage:    gotext.Get("Name of the AUR package"),
					},
					&cli.StringFlag{
						Name:    "version",
						Aliases: []string{"v"},
						Usage:   gotext.Get("Version of the package (optional, uses latest if not specified)"),
					},
				},
				Action: func(c *cli.Context) error {
					return gen.AUR(os.Stdout, gen.AUROptions{
						Name:    c.String("name"),
						Version: c.String("version"),
					})
				},
			},
		},
	}
}
