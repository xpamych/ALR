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

// execWithPrivileges выполняет команду напрямую если root или CI, иначе через sudo
func execWithPrivileges(name string, args ...string) *exec.Cmd {
	isRoot := os.Geteuid() == 0
	isCI := os.Getenv("CI") == "true"
	
	if !isRoot && !isCI {
		// Если не root и не в CI, используем sudo
		allArgs := append([]string{name}, args...)
		return exec.Command("sudo", allArgs...)
	} else {
		// Если root или в CI, запускаем напрямую
		return exec.Command(name, args...)
	}
}

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
						
						sudoCmd := execWithPrivileges("rm", "-rf", fullPath)
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
					sudoCmd := execWithPrivileges("rm", "-rf", tmpDir)
					if sudoErr := sudoCmd.Run(); sudoErr != nil {
						slog.Error(gotext.Get("Unable to remove temporary directory"), "error", err)
					}
				}
			}

			// Создаем базовый каталог /tmp/alr с владельцем root:wheel и правами 2775
			err = utils.EnsureTempDirWithRootOwner(tmpDir, 0o2775)
			if err != nil {
				slog.Warn(gotext.Get("Unable to create temporary directory"), "error", err)
			}

			// Создаем каталог dl с правами для группы wheel
			dlDir := filepath.Join(tmpDir, "dl")
			err = utils.EnsureTempDirWithRootOwner(dlDir, 0o2775)
			if err != nil {
				slog.Warn(gotext.Get("Unable to create download directory"), "error", err)
			}

			// Создаем каталог pkgs с правами для группы wheel
			pkgsDir := filepath.Join(tmpDir, "pkgs")
			err = utils.EnsureTempDirWithRootOwner(pkgsDir, 0o2775)
			if err != nil {
				slog.Warn(gotext.Get("Unable to create packages directory"), "error", err)
			}

			// Исправляем права на все существующие файлы в /tmp/alr, если там что-то есть
			if _, err := os.Stat(tmpDir); err == nil {
				slog.Info(gotext.Get("Fixing permissions on temporary files"))
				
				// Проверяем, есть ли файлы в директории
				entries, err := os.ReadDir(tmpDir)
				if err == nil && len(entries) > 0 {
					group := utils.GetPrivilegedGroup()
					fixCmd := execWithPrivileges("chown", "-R", "root:"+group, tmpDir)
					if fixErr := fixCmd.Run(); fixErr != nil {
						slog.Warn(gotext.Get("Unable to fix file ownership"), "error", fixErr)
					}
					
					fixCmd = execWithPrivileges("chmod", "-R", "2775", tmpDir)
					if fixErr := fixCmd.Run(); fixErr != nil {
						slog.Warn(gotext.Get("Unable to fix file permissions"), "error", fixErr)
					}
				}
			}

			slog.Info(gotext.Get("Rebuilding cache"))

			// Создаем директорию кэша с правильными правами
			slog.Info(gotext.Get("Creating cache directory"))
			err = utils.EnsureTempDirWithRootOwner(paths.CacheDir, 0o2775)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Unable to create new cache directory"), err)
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
