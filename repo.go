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
	"path/filepath"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	appbuilder "gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils/app_builder"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/utils"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/types"
)

func RepoCmd() *cli.Command {
	return &cli.Command{
		Name:  "repo",
		Usage: gotext.Get("Manage repos"),
		Subcommands: []*cli.Command{
			RemoveRepoCmd(),
			AddRepoCmd(),
			SetRepoRefCmd(),
		},
	}
}

func RemoveRepoCmd() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Usage:     gotext.Get("Remove an existing repository"),
		Aliases:   []string{"rm"},
		ArgsUsage: gotext.Get("<name>"),
		Action: utils.RootNeededAction(func(c *cli.Context) error {
			if c.Args().Len() < 1 {
				return cliutils.FormatCliExit("missing args", nil)
			}
			name := c.Args().Get(0)

			ctx := c.Context

			deps, err := appbuilder.
				New(ctx).
				WithConfig().
				Build()
			if err != nil {
				return err
			}
			defer deps.Defer()

			cfg := deps.Cfg

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
				return cliutils.FormatCliExit(gotext.Get("Repo \"%s\" does not exist", name), nil)
			}

			cfg.SetRepos(slices.Delete(reposSlice, index, index+1))

			err = os.RemoveAll(filepath.Join(cfg.GetPaths().RepoDir, name))
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error removing repo directory"), err)
			}
			err = cfg.SaveUserConfig()
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error saving config"), err)
			}

			if err := utils.ExitIfCantDropCapsToAlrUser(); err != nil {
				return err
			}

			deps, err = appbuilder.
				New(ctx).
				UseConfig(cfg).
				WithDB().
				Build()
			if err != nil {
				return err
			}
			defer deps.Defer()

			err = deps.DB.DeletePkgs(ctx, "repository = ?", name)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error removing packages from database"), err)
			}

			return nil
		}),
	}
}

func AddRepoCmd() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     gotext.Get("Add a new repository"),
		ArgsUsage: gotext.Get("<name> <url>"),
		Action: utils.RootNeededAction(func(c *cli.Context) error {
			if c.Args().Len() < 2 {
				return cliutils.FormatCliExit("missing args", nil)
			}

			name := c.Args().Get(0)
			repoURL := c.Args().Get(1)

			ctx := c.Context

			deps, err := appbuilder.
				New(ctx).
				WithConfig().
				Build()
			if err != nil {
				return err
			}
			defer deps.Defer()

			cfg := deps.Cfg

			reposSlice := cfg.Repos()
			for _, repo := range reposSlice {
				if repo.URL == repoURL || repo.Name == name {
					return cliutils.FormatCliExit(gotext.Get("Repo \"%s\" already exists", repo.Name), nil)
				}
			}
			reposSlice = append(reposSlice, types.Repo{
				Name: name,
				URL:  repoURL,
			})
			cfg.SetRepos(reposSlice)

			err = cfg.SaveUserConfig()
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error saving config"), err)
			}

			if err := utils.ExitIfCantDropCapsToAlrUserNoPrivs(); err != nil {
				return err
			}

			deps, err = appbuilder.
				New(ctx).
				UseConfig(cfg).
				WithDB().
				WithReposForcePull().
				Build()
			if err != nil {
				return err
			}
			defer deps.Defer()

			return nil
		}),
	}
}

func SetRepoRefCmd() *cli.Command {
	return &cli.Command{
		Name:      "set-ref",
		Usage:     gotext.Get("Set the reference of the repository"),
		ArgsUsage: gotext.Get("<name> <ref>"),
		Action: utils.RootNeededAction(func(c *cli.Context) error {
			if c.Args().Len() < 2 {
				return cliutils.FormatCliExit("missing args", nil)
			}

			name := c.Args().Get(0)
			ref := c.Args().Get(1)

			deps, err := appbuilder.
				New(c.Context).
				WithConfig().
				WithDB().
				WithReposNoPull().
				Build()
			if err != nil {
				return err
			}
			defer deps.Defer()

			repos := deps.Cfg.Repos()
			newRepos := []types.Repo{}
			for _, repo := range repos {
				if repo.Name == name {
					repo.Ref = ref
				}
				newRepos = append(newRepos, repo)
			}
			deps.Cfg.SetRepos(newRepos)
			err = deps.Cfg.SaveUserConfig()
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error saving config"), err)
			}

			err = deps.Repos.Pull(c.Context, newRepos)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error pulling repositories"), err)
			}

			return nil
		}),
	}
}

// TODO: remove
//
// Deprecated: use "alr repo add"
func LegacyAddRepoCmd() *cli.Command {
	return &cli.Command{
		Hidden:  true,
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
		Action: utils.RootNeededAction(func(c *cli.Context) error {
			cliutils.WarnLegacyCommand("alr repo add <name> <url>")
			return c.App.RunContext(c.Context, []string{"", "repo", "add", c.String("name"), c.String("url")})
		}),
	}
}

// TODO: remove
//
// Deprecated: use "alr repo rm"
func LegacyRemoveRepoCmd() *cli.Command {
	return &cli.Command{
		Hidden:  true,
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
		Action: utils.RootNeededAction(func(c *cli.Context) error {
			cliutils.WarnLegacyCommand("alr repo remove <name>")
			return c.App.RunContext(c.Context, []string{"", "repo", "remove", c.String("name")})
		}),
	}
}
