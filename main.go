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
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/leonelquinteros/gotext"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/manager"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/translations"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/logger"
)

func VersionCmd() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: gotext.Get("Print the current ALR version and exit"),
		Action: func(ctx *cli.Context) error {
			println(config.Version)
			return nil
		},
	}
}

func GetApp() *cli.App {
	return &cli.App{
		Name:  "alr",
		Usage: "Any Linux Repository",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "pm-args",
				Aliases: []string{"P"},
				Usage:   gotext.Get("Arguments to be passed on to the package manager"),
			},
			&cli.BoolFlag{
				Name:    "interactive",
				Aliases: []string{"i"},
				Value:   isatty.IsTerminal(os.Stdin.Fd()),
				Usage:   gotext.Get("Enable interactive questions and prompts"),
			},
		},
		Commands: []*cli.Command{
			InstallCmd(),
			RemoveCmd(),
			UpgradeCmd(),
			InfoCmd(),
			ListCmd(),
			BuildCmd(),
			LegacyAddRepoCmd(),
			LegacyRemoveRepoCmd(),
			RefreshCmd(),
			FixCmd(),
			GenCmd(),
			HelperCmd(),
			VersionCmd(),
			SearchCmd(),
			RepoCmd(),
			ConfigCmd(),
			// Internal commands
			InternalBuildCmd(),
			InternalInstallCmd(),
			InternalReposCmd(),
		},
		Before: func(c *cli.Context) error {
			if trimmed := strings.TrimSpace(c.String("pm-args")); trimmed != "" {
				args := strings.Split(trimmed, " ")
				manager.Args = append(manager.Args, args...)
			}
			return nil
		},
		EnableBashCompletion: true,
		ExitErrHandler: func(cCtx *cli.Context, err error) {
			cliutils.HandleExitCoder(err)
		},
	}
}

func setLogLevel(newLevel string) {
	level := slog.LevelInfo
	switch newLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	}
	logger, ok := slog.Default().Handler().(*logger.Logger)
	if !ok {
		panic("unexpected")
	}
	logger.SetLevel(level)
}

func main() {
	logger.SetupDefault()
	setLogLevel(os.Getenv("ALR_LOG_LEVEL"))
	translations.Setup()

	ctx := context.Background()

	app := GetApp()
	cfg := config.New()
	err := cfg.Load()
	if err != nil {
		slog.Error(gotext.Get("Error loading config"), "err", err)
		os.Exit(1)
	}
	setLogLevel(cfg.LogLevel())

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Make the application more internationalized
	cli.AppHelpTemplate = cliutils.GetAppCliTemplate()
	cli.CommandHelpTemplate = cliutils.GetCommandHelpTemplate()
	cli.HelpFlag.(*cli.BoolFlag).Usage = gotext.Get("Show help")

	err = app.RunContext(ctx, os.Args)
	if err != nil {
		slog.Error(gotext.Get("Error while running app"), "err", err)
	}
}
