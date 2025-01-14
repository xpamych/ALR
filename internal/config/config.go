/*
 * ALR - Any Linux Repository
 * Copyright (C) 2024 Евгений Храмов
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package config

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/pelletier/go-toml/v2"
	"plemya-x.ru/alr/internal/types"
	"plemya-x.ru/alr/pkg/loggerctx"
)

type ALRConfig struct {
	cfg   *types.Config
	paths *Paths

	cfgOnce   sync.Once
	pathsOnce sync.Once
}

func New() *ALRConfig {
	return &ALRConfig{}
}

func (c *ALRConfig) Load(ctx context.Context) {
	log := loggerctx.From(ctx)
	cfgFl, err := os.Open(c.GetPaths(ctx).ConfigPath)
	if err != nil {
		log.Warn("Error opening config file, using defaults").Err(err).Send()
		c.cfg = defaultConfig
		return
	}
	defer cfgFl.Close()

	// Copy the default configuration into config
	defCopy := *defaultConfig
	config := &defCopy
	config.Repos = nil

	err = toml.NewDecoder(cfgFl).Decode(config)
	if err != nil {
		log.Warn("Error decoding config file, using defaults").Err(err).Send()
		c.cfg = defaultConfig
		return
	}
	c.cfg = config
}

func (c *ALRConfig) initPaths(ctx context.Context) {
	log := loggerctx.From(ctx)
	paths := &Paths{}

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("Unable to detect user config directory").Err(err).Send()
	}

	paths.ConfigDir = filepath.Join(cfgDir, "alr")

	err = os.MkdirAll(paths.ConfigDir, 0o755)
	if err != nil {
		log.Fatal("Unable to create ALR config directory").Err(err).Send()
	}

	paths.ConfigPath = filepath.Join(paths.ConfigDir, "alr.toml")

	if _, err := os.Stat(paths.ConfigPath); err != nil {
		cfgFl, err := os.Create(paths.ConfigPath)
		if err != nil {
			log.Fatal("Unable to create ALR config file").Err(err).Send()
		}

		err = toml.NewEncoder(cfgFl).Encode(&defaultConfig)
		if err != nil {
			log.Fatal("Error encoding default configuration").Err(err).Send()
		}

		cfgFl.Close()
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatal("Unable to detect cache directory").Err(err).Send()
	}

	paths.CacheDir = filepath.Join(cacheDir, "alr")
	paths.RepoDir = filepath.Join(paths.CacheDir, "repo")
	paths.PkgsDir = filepath.Join(paths.CacheDir, "pkgs")

	err = os.MkdirAll(paths.RepoDir, 0o755)
	if err != nil {
		log.Fatal("Unable to create repo cache directory").Err(err).Send()
	}

	err = os.MkdirAll(paths.PkgsDir, 0o755)
	if err != nil {
		log.Fatal("Unable to create package cache directory").Err(err).Send()
	}

	paths.DBPath = filepath.Join(paths.CacheDir, "db")

	c.paths = paths
}

func (c *ALRConfig) GetPaths(ctx context.Context) *Paths {
	c.pathsOnce.Do(func() {
		c.initPaths(ctx)
	})
	return c.paths
}

func (c *ALRConfig) Repos(ctx context.Context) []types.Repo {
	c.cfgOnce.Do(func() {
		c.Load(ctx)
	})
	return c.cfg.Repos
}

func (c *ALRConfig) IgnorePkgUpdates(ctx context.Context) []string {
	c.cfgOnce.Do(func() {
		c.Load(ctx)
	})
	return c.cfg.IgnorePkgUpdates
}
