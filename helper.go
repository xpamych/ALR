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
	"strings"

	"github.com/urfave/cli/v2"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"

	"plemya-x.ru/alr/internal/cpu"
	"plemya-x.ru/alr/internal/shutils/helpers"
	"plemya-x.ru/alr/pkg/distro"
	"plemya-x.ru/alr/pkg/loggerctx"
)

var helperCmd = &cli.Command{
	Name:        "helper",
	Usage:       "Run a ALR helper command",
	ArgsUsage:   `<helper_name|"list">`,
	Subcommands: []*cli.Command{helperListCmd},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "dest-dir",
			Aliases: []string{"d"},
			Usage:   "The directory that the install commands will install to",
			Value:   "dest",
		},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		log := loggerctx.From(ctx)

		if c.Args().Len() < 1 {
			cli.ShowSubcommandHelpAndExit(c, 1)
		}

		helper, ok := helpers.Helpers[c.Args().First()]
		if !ok {
			log.Fatal("No such helper command").Str("name", c.Args().First()).Send()
		}

		wd, err := os.Getwd()
		if err != nil {
			log.Fatal("Error getting working directory").Err(err).Send()
		}

		info, err := distro.ParseOSRelease(ctx)
		if err != nil {
			log.Fatal("Error getting working directory").Err(err).Send()
		}

		hc := interp.HandlerContext{
			Env: expand.ListEnviron(
				"pkgdir="+c.String("dest-dir"),
				"DISTRO_ID="+info.ID,
				"DISTRO_ID_LIKE="+strings.Join(info.Like, " "),
				"ARCH="+cpu.Arch(),
			),
			Dir:    wd,
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		return helper(hc, c.Args().First(), c.Args().Slice()[1:])
	},
	CustomHelpTemplate: cli.CommandHelpTemplate,
	BashComplete: func(ctx *cli.Context) {
		for name := range helpers.Helpers {
			fmt.Println(name)
		}
	},
}

var helperListCmd = &cli.Command{
	Name:    "list",
	Usage:   "List all the available helper commands",
	Aliases: []string{"ls"},
	Action: func(ctx *cli.Context) error {
		for name := range helpers.Helpers {
			fmt.Println(name)
		}
		return nil
	},
}
