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

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	database "gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/repos"
)

func AddRepoCmd() *cli.Command {
	return &cli.Command{
		Name:    "addrepo",
		Usage:   gotext.Get("Add a new repository"),
		Aliases: []string{"ar"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Required: true,
				Usage:    gotext.Get("Name of the new repo"),
			},
			&cli.StringFlag{
				Name:     "url",
				Aliases:  []string{"u"},
				Required: true,
				Usage:    gotext.Get("URL of the new repo"),
			},
		},
		Action: func(c *cli.Context) error {
			ctx := c.Context

			name := c.String("name")
			repoURL := c.String("url")

			cfg := config.New()
			err := cfg.Load()
			if err != nil {
				slog.Error(gotext.Get("Error loading config"), "err", err)
				os.Exit(1)
			}

			reposSlice := cfg.Repos()

			for _, repo := range reposSlice {
				if repo.URL == repoURL {
					slog.Error("Repo already exists", "name", repo.Name)
					os.Exit(1)
				}
			}

			reposSlice = append(reposSlice, types.Repo{
				Name: name,
				URL:  repoURL,
			})
			cfg.SetRepos(reposSlice)

			err = cfg.SaveUserConfig()
			if err != nil {
				slog.Error(gotext.Get("Error saving config"), "err", err)
				os.Exit(1)
			}

			db := database.New(cfg)
			err = db.Init(ctx)
			if err != nil {
				slog.Error(gotext.Get("Error pulling repos"), "err", err)
			}

			rs := repos.New(cfg, db)
			err = rs.Pull(ctx, cfg.Repos())
			if err != nil {
				slog.Error(gotext.Get("Error pulling repos"), "err", err)
				os.Exit(1)
			}

			return nil
		},
	}
}

func RemoveRepoCmd() *cli.Command {
	return &cli.Command{
		Name:    "removerepo",
		Usage:   gotext.Get("Remove an existing repository"),
		Aliases: []string{"rr"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Required: true,
				Usage:    gotext.Get("Name of the repo to be deleted"),
			},
		},
		Action: func(c *cli.Context) error {
			ctx := c.Context

			name := c.String("name")
			cfg := config.New()
			err := cfg.Load()
			if err != nil {
				slog.Error(gotext.Get("Error loading config"), "err", err)
				os.Exit(1)
			}

			found := false
			index := 0
			reposSlice := cfg.Repos()
			for i, repo := range reposSlice {
				if repo.Name == name {
					index = i
					found = true
				}
			}
			if !found {
				slog.Error(gotext.Get("Repo does not exist"), "name", name)
				os.Exit(1)
			}

			cfg.SetRepos(slices.Delete(reposSlice, index, index+1))

			err = os.RemoveAll(filepath.Join(cfg.GetPaths().RepoDir, name))
			if err != nil {
				slog.Error(gotext.Get("Error removing repo directory"), "err", err)
				os.Exit(1)
			}

			err = cfg.SaveUserConfig()
			if err != nil {
				slog.Error(gotext.Get("Error saving config"), "err", err)
				os.Exit(1)
			}

			db := database.New(cfg)
			err = db.Init(ctx)
			if err != nil {
				os.Exit(1)
			}
			err = db.DeletePkgs(ctx, "repository = ?", name)
			if err != nil {
				slog.Error(gotext.Get("Error removing packages from database"), "err", err)
				os.Exit(1)
			}

			return nil
		},
	}
}

func RefreshCmd() *cli.Command {
	return &cli.Command{
		Name:    "refresh",
		Usage:   gotext.Get("Pull all repositories that have changed"),
		Aliases: []string{"ref"},
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
				os.Exit(1)
			}
			rs := repos.New(cfg, db)
			err = rs.Pull(ctx, cfg.Repos())
			if err != nil {
				slog.Error(gotext.Get("Error pulling repos"), "err", err)
				os.Exit(1)
			}
			return nil
		},
	}
}
