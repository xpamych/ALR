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
	"github.com/pelletier/go-toml/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/repos"
)

var addrepoCmd = &cli.Command{
	Name:    "addrepo",
	Usage:   "Add a new repository",
	Aliases: []string{"ar"},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Aliases:  []string{"n"},
			Required: true,
			Usage:    "Name of the new repo",
		},
		&cli.StringFlag{
			Name:     "url",
			Aliases:  []string{"u"},
			Required: true,
			Usage:    "URL of the new repo",
		},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context

		name := c.String("name")
		repoURL := c.String("url")

		cfg := config.Config(ctx)

		for _, repo := range cfg.Repos {
			if repo.URL == repoURL {
				slog.Error("Repo already exists", "name", repo.Name)
				os.Exit(1)
			}
		}

		cfg.Repos = append(cfg.Repos, types.Repo{
			Name: name,
			URL:  repoURL,
		})

		cfgFl, err := os.Create(config.GetPaths(ctx).ConfigPath)
		if err != nil {
			slog.Error(gotext.Get("Error opening config file"), "err", err)
			os.Exit(1)
		}

		err = toml.NewEncoder(cfgFl).Encode(cfg)
		if err != nil {
			slog.Error(gotext.Get("Error encoding config"), "err", err)
			os.Exit(1)
		}

		err = repos.Pull(ctx, cfg.Repos)
		if err != nil {
			slog.Error(gotext.Get("Error pulling repos"), "err", err)
			os.Exit(1)
		}

		return nil
	},
}

var removerepoCmd = &cli.Command{
	Name:    "removerepo",
	Usage:   "Remove an existing repository",
	Aliases: []string{"rr"},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "name",
			Aliases:  []string{"n"},
			Required: true,
			Usage:    "Name of the repo to be deleted",
		},
	},
	Action: func(c *cli.Context) error {
		ctx := c.Context

		name := c.String("name")
		cfg := config.Config(ctx)

		found := false
		index := 0
		for i, repo := range cfg.Repos {
			if repo.Name == name {
				index = i
				found = true
			}
		}
		if !found {
			slog.Error(gotext.Get("Repo does not exist"), "name", name)
			os.Exit(1)
		}

		cfg.Repos = slices.Delete(cfg.Repos, index, index+1)

		cfgFl, err := os.Create(config.GetPaths(ctx).ConfigPath)
		if err != nil {
			slog.Error(gotext.Get("Error opening config file"), "err", err)
			os.Exit(1)
		}

		err = toml.NewEncoder(cfgFl).Encode(&cfg)
		if err != nil {
			slog.Error(gotext.Get("Error encoding config"), "err", err)
			os.Exit(1)
		}

		err = os.RemoveAll(filepath.Join(config.GetPaths(ctx).RepoDir, name))
		if err != nil {
			slog.Error(gotext.Get("Error removing repo directory"), "err", err)
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

var refreshCmd = &cli.Command{
	Name:    "refresh",
	Usage:   "Pull all repositories that have changed",
	Aliases: []string{"ref"},
	Action: func(c *cli.Context) error {
		ctx := c.Context
		err := repos.Pull(ctx, config.Config(ctx).Repos)
		if err != nil {
			slog.Error(gotext.Get("Error pulling repos"), "err", err)
			os.Exit(1)
		}
		return nil
	},
}
