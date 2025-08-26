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
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	appbuilder "gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils/app_builder"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/utils"
)

func FixCmd() *cli.Command {
	return &cli.Command{
		Name:  "fix",
		Usage: gotext.Get("Attempt to fix problems with ALR"),
		Action: func(c *cli.Context) error {
			// Команда выполняется от текущего пользователя
			// При необходимости будет запрошен sudo для удаления файлов root

			ctx := c.Context

			deps, err := appbuilder.
				New(ctx).
				WithConfig().
				Build()
			if err != nil {
				return cli.Exit(err, 1)
			}
			defer deps.Defer()

			cfg := deps.Cfg

			paths := cfg.GetPaths()

			slog.Info(gotext.Get("Clearing cache and temporary directories"))

			// Проверяем, существует ли директория кэша
			dir, err := os.Open(paths.CacheDir)
			if err != nil {
				if os.IsNotExist(err) {
					// Директория не существует, просто создадим её позже
					slog.Info(gotext.Get("Cache directory does not exist, will create it"))
				} else {
					return cliutils.FormatCliExit(gotext.Get("Unable to open cache directory"), err)
				}
			} else {
				defer dir.Close()

				entries, err := dir.Readdirnames(-1)
				if err != nil {
					return cliutils.FormatCliExit(gotext.Get("Unable to read cache directory contents"), err)
				}

				for _, entry := range entries {
					fullPath := filepath.Join(paths.CacheDir, entry)

					// Пробуем сделать файлы доступными для записи
					if err := makeWritableRecursive(fullPath); err != nil {
						slog.Debug("Failed to make path writable", "path", fullPath, "error", err)
					}

					// Пробуем удалить
					err = os.RemoveAll(fullPath)
					if err != nil {
						// Если не получилось удалить, пробуем через sudo
						slog.Warn(gotext.Get("Unable to remove cache item (%s) as current user, trying with sudo", entry))
						
						sudoCmd := exec.Command("sudo", "rm", "-rf", fullPath)
						if sudoErr := sudoCmd.Run(); sudoErr != nil {
							// Если и через sudo не получилось, пропускаем с предупреждением
							slog.Error(gotext.Get("Unable to remove cache item (%s)", entry), "error", err)
							continue
						}
					}
				}
			}

			// Очищаем временные директории
			slog.Info(gotext.Get("Clearing temporary directory"))
			tmpDir := "/tmp/alr"
			if _, err := os.Stat(tmpDir); err == nil {
				// Директория существует, пробуем очистить
				err = os.RemoveAll(tmpDir)
				if err != nil {
					// Если не получилось удалить, пробуем через sudo
					slog.Warn(gotext.Get("Unable to remove temporary directory as current user, trying with sudo"))
					sudoCmd := exec.Command("sudo", "rm", "-rf", tmpDir)
					if sudoErr := sudoCmd.Run(); sudoErr != nil {
						slog.Error(gotext.Get("Unable to remove temporary directory"), "error", err)
					}
				}
			}

			// Создаем базовый каталог /tmp/alr с владельцем root:wheel и правами 775
			err = utils.EnsureTempDirWithRootOwner(tmpDir, 0o775)
			if err != nil {
				slog.Warn(gotext.Get("Unable to create temporary directory"), "error", err)
			}

			// Создаем каталог dl с правами для группы wheel
			dlDir := filepath.Join(tmpDir, "dl")
			err = utils.EnsureTempDirWithRootOwner(dlDir, 0o775)
			if err != nil {
				slog.Warn(gotext.Get("Unable to create download directory"), "error", err)
			}

			// Создаем каталог pkgs с правами для группы wheel
			pkgsDir := filepath.Join(tmpDir, "pkgs")
			err = utils.EnsureTempDirWithRootOwner(pkgsDir, 0o775)
			if err != nil {
				slog.Warn(gotext.Get("Unable to create packages directory"), "error", err)
			}

			// Исправляем права на все существующие файлы в /tmp/alr, если там что-то есть
			if _, err := os.Stat(tmpDir); err == nil {
				slog.Info(gotext.Get("Fixing permissions on temporary files"))
				
				// Проверяем, есть ли файлы в директории
				entries, err := os.ReadDir(tmpDir)
				if err == nil && len(entries) > 0 {
					fixCmd := exec.Command("sudo", "chown", "-R", "root:wheel", tmpDir)
					if fixErr := fixCmd.Run(); fixErr != nil {
						slog.Warn(gotext.Get("Unable to fix file ownership"), "error", fixErr)
					}
					
					fixCmd = exec.Command("sudo", "chmod", "-R", "2775", tmpDir)
					if fixErr := fixCmd.Run(); fixErr != nil {
						slog.Warn(gotext.Get("Unable to fix file permissions"), "error", fixErr)
					}
				}
			}

			slog.Info(gotext.Get("Rebuilding cache"))

			// Пробуем создать директорию кэша
			err = os.MkdirAll(paths.CacheDir, 0o775)
			if err != nil {
				// Если не получилось, пробуем через sudo с правильными правами для группы wheel
				slog.Info(gotext.Get("Creating cache directory with sudo"))
				sudoCmd := exec.Command("sudo", "mkdir", "-p", paths.CacheDir)
				if sudoErr := sudoCmd.Run(); sudoErr != nil {
					return cliutils.FormatCliExit(gotext.Get("Unable to create new cache directory"), err)
				}
				
				// Устанавливаем права 775 и группу wheel
				chmodCmd := exec.Command("sudo", "chmod", "775", paths.CacheDir)
				if chmodErr := chmodCmd.Run(); chmodErr != nil {
					return cliutils.FormatCliExit(gotext.Get("Unable to set cache directory permissions"), chmodErr)
				}
				
				chgrpCmd := exec.Command("sudo", "chgrp", "wheel", paths.CacheDir)
				if chgrpErr := chgrpCmd.Run(); chgrpErr != nil {
					return cliutils.FormatCliExit(gotext.Get("Unable to set cache directory group"), chgrpErr)
				}
			}

			deps, err = appbuilder.
				New(ctx).
				WithConfig().
				WithDB().
				WithReposForcePull().
				Build()
			if err != nil {
				return cli.Exit(err, 1)
			}
			defer deps.Defer()

			slog.Info(gotext.Get("Done"))

			return nil
		},
	}
}

func makeWritableRecursive(path string) error {
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		newMode := info.Mode() | 0o200
		if d.IsDir() {
			newMode |= 0o100
		}

		return os.Chmod(path, newMode)
	})
}
