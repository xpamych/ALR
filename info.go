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
	"os"

	"github.com/goccy/go-yaml"
	"github.com/jeandeaual/go-locale"
	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	appbuilder "gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils/app_builder"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/overrides"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/utils"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/alrsh"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/distro"
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
		BashComplete: cliutils.BashCompleteWithError(func(c *cli.Context) error {
			if err := utils.ExitIfCantDropCapsToAlrUser(); err != nil {
				return err
			}

			ctx := c.Context
			deps, err := appbuilder.
				New(ctx).
				WithConfig().
				WithDB().
				Build()
			if err != nil {
				return err
			}
			defer deps.Defer()

			result, err := deps.DB.GetPkgs(c.Context, "true")
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error getting packages"), err)
			}

			for _, pkg := range result {
				fmt.Println(pkg.Name)
			}
			return nil
		}),
		Action: func(c *cli.Context) error {
			if err := utils.ExitIfCantDropCapsToAlrUserNoPrivs(); err != nil {
				return err
			}

			args := c.Args()
			if args.Len() < 1 {
				return cli.Exit(gotext.Get("Command info expected at least 1 argument, got %d", args.Len()), 1)
			}

			ctx := c.Context

			deps, err := appbuilder.
				New(ctx).
				WithConfig().
				WithDB().
				WithDistroInfo().
				WithRepos().
				Build()
			if err != nil {
				return cli.Exit(err, 1)
			}
			defer deps.Defer()

			rs := deps.Repos

			found, _, err := rs.FindPkgs(ctx, args.Slice())
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error finding packages"), err)
			}

			if len(found) == 0 {
				return cliutils.FormatCliExit(gotext.Get("Package not found"), err)
			}

			pkgs := cliutils.FlattenPkgs(ctx, found, "show", c.Bool("interactive"))

			var names []string
			all := c.Bool("all")

			systemLang, err := locale.GetLanguage()
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Can't detect system language"), err)
			}
			if systemLang == "" {
				systemLang = "en"
			}

			info, err := distro.ParseOSRelease(ctx)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error parsing os-release file"), err)
			}
			names, err = overrides.Resolve(
				info,
				overrides.DefaultOpts.
					WithLanguages([]string{systemLang}),
			)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error resolving overrides"), err)
			}

			for _, pkg := range pkgs {
				alrsh.ResolvePackage(&pkg, names)
				view := alrsh.NewPackageView(pkg)
				view.Resolved = !all
				err = yaml.NewEncoder(os.Stdout, yaml.UseJSONMarshaler(), yaml.OmitEmpty()).Encode(view)
				if err != nil {
					return cliutils.FormatCliExit(gotext.Get("Error encoding script variables"), err)
				}
				fmt.Println("---")
			}

			return nil
		},
	}
}
