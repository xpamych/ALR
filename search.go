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
	"text/template"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	appbuilder "gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils/app_builder"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/utils"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/search"
)

func SearchCmd() *cli.Command {
	return &cli.Command{
		Name:    "search",
		Usage:   gotext.Get("Search packages"),
		Aliases: []string{"s"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   gotext.Get("Search by name"),
			},
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   gotext.Get("Search by description"),
			},
			&cli.StringFlag{
				Name:    "repository",
				Aliases: []string{"repo"},
				Usage:   gotext.Get("Search by repository"),
			},
			&cli.StringFlag{
				Name:    "provides",
				Aliases: []string{"p"},
				Usage:   gotext.Get("Search by provides"),
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   gotext.Get("Format output using a Go template"),
			},
		},
		Action: func(c *cli.Context) error {
			if err := utils.ExitIfCantDropCapsToAlrUserNoPrivs(); err != nil {
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

			db := deps.DB

			s := search.New(db)

			packages, err := s.Search(
				ctx,
				search.NewSearchOptions().
					WithName(c.String("name")).
					WithDescription(c.String("description")).
					WithRepository(c.String("repository")).
					WithProvides(c.String("provides")).
					Build(),
			)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error while executing search"), err)
			}

			format := c.String("format")
			var tmpl *template.Template
			if format != "" {
				tmpl, err = template.New("format").Parse(format)
				if err != nil {
					return cliutils.FormatCliExit(gotext.Get("Error parsing format template"), err)
				}
			}

			for _, dbPkg := range packages {
				if tmpl != nil {
					err = tmpl.Execute(os.Stdout, dbPkg)
					if err != nil {
						return cliutils.FormatCliExit(gotext.Get("Error executing template"), err)
					}
					fmt.Println()
				} else {
					fmt.Println(dbPkg.Name)
				}
			}

			return nil
		},
	}
}
