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
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/constants"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/types"
)

type ALRConfig struct {
	cfg   *types.Config
	paths *Paths

	System *SystemConfig
	env    *EnvConfig
}

func New() *ALRConfig {
	return &ALRConfig{
		System: NewSystemConfig(),
		env:    NewEnvConfig(),
	}
}

func defaultConfigKoanf() *koanf.Koanf {
	k := koanf.New(".")
	defaults := map[string]interface{}{
		"rootCmd":          "sudo",
		"useRootCmd":       true,
		"pagerStyle":       "native",
		"ignorePkgUpdates": []string{},
		"logLevel":         "info",
		"autoPull":         true,
		"repos": []types.Repo{
			{
				Name: "alr-default",
				URL:  "https://gitea.plemya-x.ru/Plemya-x/alr-default.git",
			},
		},
	}
	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		panic(k)
	}
	return k
}

func (c *ALRConfig) Load() error {
	config := types.Config{}

	merged := koanf.New(".")

	if err := c.System.Load(); err != nil {
		return fmt.Errorf("failed to load system config: %w", err)
	}

	if err := c.env.Load(); err != nil {
		return fmt.Errorf("failed to load env config: %w", err)
	}

	systemK := c.System.koanf()
	envK := c.env.koanf()

	if err := merged.Merge(defaultConfigKoanf()); err != nil {
		return fmt.Errorf("failed to merge default config: %w", err)
	}
	if err := merged.Merge(systemK); err != nil {
		return fmt.Errorf("failed to merge system config: %w", err)
	}
	if err := merged.Merge(envK); err != nil {
		return fmt.Errorf("failed to merge env config: %w", err)
	}
	if err := merged.Unmarshal("", &config); err != nil {
		return fmt.Errorf("failed to unmarshal merged config: %w", err)
	}

	c.cfg = &config

	c.paths = &Paths{}
	c.paths.UserConfigPath = constants.SystemConfigPath
	c.paths.CacheDir = constants.SystemCachePath
	c.paths.RepoDir = filepath.Join(c.paths.CacheDir, "repo")
	c.paths.PkgsDir = filepath.Join(constants.TempDir, "pkgs")  // Перемещаем в /tmp/alr/pkgs
	c.paths.DBPath = filepath.Join(c.paths.CacheDir, "alr.db")

	// Проверяем существование кэш-директории, но не пытаемся создать
	if _, err := os.Stat(c.paths.CacheDir); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to check cache directory: %w", err)
		}
	}

	return nil
}

func (c *ALRConfig) ToYAML() (string, error) {
	data, err := yaml.Marshal(c.cfg)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *ALRConfig) RootCmd() string             { return c.cfg.RootCmd }
func (c *ALRConfig) PagerStyle() string          { return c.cfg.PagerStyle }
func (c *ALRConfig) AutoPull() bool              { return c.cfg.AutoPull }
func (c *ALRConfig) Repos() []types.Repo         { return c.cfg.Repos }
func (c *ALRConfig) SetRepos(repos []types.Repo) { c.System.SetRepos(repos) }
func (c *ALRConfig) IgnorePkgUpdates() []string  { return c.cfg.IgnorePkgUpdates }
func (c *ALRConfig) LogLevel() string            { return c.cfg.LogLevel }
func (c *ALRConfig) UseRootCmd() bool            { return c.cfg.UseRootCmd }
func (c *ALRConfig) GetPaths() *Paths            { return c.paths }
