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
	"log/slog"
	"os"
	"syscall"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/build"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	appbuilder "gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils/app_builder"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/logger"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/manager"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/utils"
)

func InternalBuildCmd() *cli.Command {
	return &cli.Command{
		Name:     "_internal-safe-script-executor",
		HideHelp: true,
		Hidden:   true,
		Action: func(c *cli.Context) error {
			logger.SetupForGoPlugin()

			slog.Debug("start _internal-safe-script-executor", "uid", syscall.Getuid(), "gid", syscall.Getgid())


			cfg := config.New()
			err := cfg.Load()
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Error loading config"), err)
			}

			logger := hclog.New(&hclog.LoggerOptions{
				Name:        "plugin",
				Output:      os.Stderr,
				Level:       hclog.Debug,
				JSONFormat:  false,
				DisableTime: true,
			})

			plugin.Serve(&plugin.ServeConfig{
				HandshakeConfig: build.HandshakeConfig,
				Plugins: map[string]plugin.Plugin{
					"script-executor": &build.ScriptExecutorPlugin{
						Impl: build.NewLocalScriptExecutor(cfg),
					},
				},
				Logger: logger,
			})
			return nil
		},
	}
}

func InternalReposCmd() *cli.Command {
	return &cli.Command{
		Name:     "_internal-repos",
		HideHelp: true,
		Hidden:   true,
		Action: utils.RootNeededAction(func(ctx *cli.Context) error {
			logger.SetupForGoPlugin()


			deps, err := appbuilder.
				New(ctx.Context).
				WithConfig().
				WithDB().
				WithReposNoPull().
				Build()
			if err != nil {
				return err
			}
			defer deps.Defer()

			pluginCfg := build.GetPluginServeCommonConfig()
			pluginCfg.Plugins = map[string]plugin.Plugin{
				"repos": &build.ReposExecutorPlugin{
					Impl: build.NewRepos(
						deps.Repos,
					),
				},
			}
			plugin.Serve(pluginCfg)
			return nil
		}),
	}
}

func InternalInstallCmd() *cli.Command {
	return &cli.Command{
		Name:     "_internal-installer",
		HideHelp: true,
		Hidden:   true,
		Action: func(c *cli.Context) error {
			logger.SetupForGoPlugin()

			// Запуск от текущего пользователя, повышение прав будет через sudo при необходимости

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

			logger := hclog.New(&hclog.LoggerOptions{
				Name:        "plugin",
				Output:      os.Stderr,
				Level:       hclog.Trace,
				JSONFormat:  true,
				DisableTime: true,
			})

			plugin.Serve(&plugin.ServeConfig{
				HandshakeConfig: build.HandshakeConfig,
				Plugins: map[string]plugin.Plugin{
					"installer": &build.InstallerExecutorPlugin{
						Impl: build.NewInstaller(
							manager.Detect(),
						),
					},
				},
				Logger: logger,
			})
			return nil
		},
	}
}


