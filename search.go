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
	"text/template"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	database "gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
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
			ctx := c.Context
			cfg := config.New()
			db := database.New(cfg)
			err := db.Init(ctx)
			defer db.Close()

			if err != nil {
				slog.Error(gotext.Get("Error initialization database"), "err", err)
				os.Exit(1)
			}

			format := c.String("format")
			var tmpl *template.Template
			if format != "" {
				tmpl, err = template.New("format").Parse(format)
				if err != nil {
					slog.Error(gotext.Get("Error parsing format template"), "err", err)
					os.Exit(1)
				}
			}

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
				slog.Error(gotext.Get("Error parsing format template"), "err", err)
				os.Exit(1)
			}

			for _, dbPkg := range packages {
				if tmpl != nil {
					err = tmpl.Execute(os.Stdout, dbPkg)
					if err != nil {
						slog.Error(gotext.Get("Error executing template"), "err", err)
						os.Exit(1)
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
