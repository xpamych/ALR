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

package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"reflect"

	"github.com/caarlos0/env"
	"github.com/pelletier/go-toml/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/constants"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/types"
)

type ALRConfig struct {
	cfg   *types.Config
	paths *Paths
}

var defaultConfig = &types.Config{
	RootCmd:          "sudo",
	UseRootCmd:       true,
	PagerStyle:       "native",
	IgnorePkgUpdates: []string{},
	AutoPull:         true,
	Repos:            []types.Repo{},
}

func New() *ALRConfig {
	return &ALRConfig{}
}

func readConfig(path string) (*types.Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := types.Config{}

	if err := toml.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func mergeStructs(dst, src interface{}) {
	srcVal := reflect.ValueOf(src)
	if srcVal.IsNil() {
		return
	}
	srcVal = srcVal.Elem()
	dstVal := reflect.ValueOf(dst).Elem()

	for i := range srcVal.NumField() {
		srcField := srcVal.Field(i)
		srcFieldName := srcVal.Type().Field(i).Name

		dstField := dstVal.FieldByName(srcFieldName)
		if dstField.IsValid() && dstField.CanSet() {
			dstField.Set(srcField)
		}
	}
}

func (c *ALRConfig) Load() error {
	systemConfig, err := readConfig(
		constants.SystemConfigPath,
	)
	if err != nil {
		slog.Debug("Cannot read system config", "err", err)
	}

	config := &types.Config{}

	mergeStructs(config, defaultConfig)
	mergeStructs(config, systemConfig)
	err = env.Parse(config)
	if err != nil {
		return err
	}

	c.cfg = config

	c.paths = &Paths{}
	c.paths.UserConfigPath = constants.SystemConfigPath
	c.paths.CacheDir = constants.SystemCachePath
	c.paths.RepoDir = filepath.Join(c.paths.CacheDir, "repo")
	c.paths.PkgsDir = filepath.Join(c.paths.CacheDir, "pkgs")
	c.paths.DBPath = filepath.Join(c.paths.CacheDir, "db")
	// c.initPaths()

	return nil
}

func (c *ALRConfig) RootCmd() string {
	return c.cfg.RootCmd
}

func (c *ALRConfig) PagerStyle() string {
	return c.cfg.PagerStyle
}

func (c *ALRConfig) AutoPull() bool {
	return c.cfg.AutoPull
}

func (c *ALRConfig) Repos() []types.Repo {
	return c.cfg.Repos
}

func (c *ALRConfig) SetRepos(repos []types.Repo) {
	c.cfg.Repos = repos
}

func (c *ALRConfig) IgnorePkgUpdates() []string {
	return c.cfg.IgnorePkgUpdates
}

func (c *ALRConfig) LogLevel() string {
	return c.cfg.LogLevel
}

func (c *ALRConfig) UseRootCmd() bool {
	return c.cfg.UseRootCmd
}

func (c *ALRConfig) GetPaths() *Paths {
	return c.paths
}

func (c *ALRConfig) SaveUserConfig() error {
	f, err := os.Create(c.paths.UserConfigPath)
	if err != nil {
		return err
	}

	return toml.NewEncoder(f).Encode(c.cfg)
}
