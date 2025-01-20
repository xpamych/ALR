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
	"os"

	"github.com/urfave/cli/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/loggerctx"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/repos"
)

var fixCmd = &cli.Command{
	Name:  "fix",
	Usage: "Attempt to fix problems with ALR",
	Action: func(c *cli.Context) error {
		ctx := c.Context
		log := loggerctx.From(ctx)

		db.Close()
		paths := config.GetPaths(ctx)

		log.Info("Removing cache directory").Send()

		err := os.RemoveAll(paths.CacheDir)
		if err != nil {
			log.Fatal("Unable to remove cache directory").Err(err).Send()
		}

		log.Info("Rebuilding cache").Send()

		err = os.MkdirAll(paths.CacheDir, 0o755)
		if err != nil {
			log.Fatal("Unable to create new cache directory").Err(err).Send()
		}

		err = repos.Pull(ctx, config.Config(ctx).Repos)
		if err != nil {
			log.Fatal("Error pulling repos").Err(err).Send()
		}

		log.Info("Done").Send()

		return nil
	},
}
