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

package config

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/pelletier/go-toml/v2"

	"github.com/leonelquinteros/gotext"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
)

type ALRConfig struct {
	cfg   *types.Config
	paths *Paths

	cfgOnce   sync.Once
	pathsOnce sync.Once
}

var defaultConfig = &types.Config{
	RootCmd:          "sudo",
	PagerStyle:       "native",
	IgnorePkgUpdates: []string{},
	AutoPull:         false,
	Repos:            []types.Repo{},
}

func New() *ALRConfig {
	return &ALRConfig{}
}

func (c *ALRConfig) Load(ctx context.Context) {
	cfgFl, err := os.Open(c.GetPaths(ctx).ConfigPath)
	if err != nil {
		slog.Warn(gotext.Get("Error opening config file, using defaults"), "err", err)
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
		slog.Warn(gotext.Get("Error decoding config file, using defaults"), "err", err)
		c.cfg = defaultConfig
		return
	}
	c.cfg = config
}

func (c *ALRConfig) initPaths() {
	paths := &Paths{}

	cfgDir, err := os.UserConfigDir()
	if err != nil {
		slog.Error(gotext.Get("Unable to detect user config directory"), "err", err)
		os.Exit(1)
	}

	paths.ConfigDir = filepath.Join(cfgDir, "alr")

	err = os.MkdirAll(paths.ConfigDir, 0o755)
	if err != nil {
		slog.Error(gotext.Get("Unable to create ALR config directory"), "err", err)
		os.Exit(1)
	}

	paths.ConfigPath = filepath.Join(paths.ConfigDir, "alr.toml")

	if _, err := os.Stat(paths.ConfigPath); err != nil {
		cfgFl, err := os.Create(paths.ConfigPath)
		if err != nil {
			slog.Error(gotext.Get("Unable to create ALR config file"), "err", err)
			os.Exit(1)
		}

		err = toml.NewEncoder(cfgFl).Encode(&defaultConfig)
		if err != nil {
			slog.Error(gotext.Get("Error encoding default configuration"), "err", err)
			os.Exit(1)
		}

		cfgFl.Close()
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		slog.Error(gotext.Get("Unable to detect cache directory"), "err", err)
		os.Exit(1)
	}

	paths.CacheDir = filepath.Join(cacheDir, "alr")
	paths.RepoDir = filepath.Join(paths.CacheDir, "repo")
	paths.PkgsDir = filepath.Join(paths.CacheDir, "pkgs")

	err = os.MkdirAll(paths.RepoDir, 0o755)
	if err != nil {
		slog.Error(gotext.Get("Unable to create repo cache directory"), "err", err)
		os.Exit(1)
	}

	err = os.MkdirAll(paths.PkgsDir, 0o755)
	if err != nil {
		slog.Error(gotext.Get("Unable to create package cache directory"), "err", err)
		os.Exit(1)
	}

	paths.DBPath = filepath.Join(paths.CacheDir, "db")

	c.paths = paths
}

func (c *ALRConfig) GetPaths(ctx context.Context) *Paths {
	c.pathsOnce.Do(func() {
		c.initPaths()
	})
	return c.paths
}

func (c *ALRConfig) Repos(ctx context.Context) []types.Repo {
	c.cfgOnce.Do(func() {
		c.Load(ctx)
	})
	return c.cfg.Repos
}

func (c *ALRConfig) SetRepos(ctx context.Context, repos []types.Repo) {
	c.cfgOnce.Do(func() {
		c.Load(ctx)
	})
	c.cfg.Repos = repos
}

func (c *ALRConfig) IgnorePkgUpdates(ctx context.Context) []string {
	c.cfgOnce.Do(func() {
		c.Load(ctx)
	})
	return c.cfg.IgnorePkgUpdates
}

func (c *ALRConfig) AutoPull(ctx context.Context) bool {
	c.cfgOnce.Do(func() {
		c.Load(ctx)
	})
	return c.cfg.AutoPull
}

func (c *ALRConfig) PagerStyle(ctx context.Context) string {
	c.cfgOnce.Do(func() {
		c.Load(ctx)
	})
	return c.cfg.PagerStyle
}

func (c *ALRConfig) AllowRunAsRoot(ctx context.Context) bool {
	c.cfgOnce.Do(func() {
		c.Load(ctx)
	})
	return c.cfg.Unsafe.AllowRunAsRoot
}

func (c *ALRConfig) RootCmd(ctx context.Context) string {
	c.cfgOnce.Do(func() {
		c.Load(ctx)
	})
	return c.cfg.RootCmd
}
