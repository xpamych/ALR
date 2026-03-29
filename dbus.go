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
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"

	"git.alr-pkg.ru/Plemya-x/ALR/internal/cliutils"
	alrdbus "git.alr-pkg.ru/Plemya-x/ALR/internal/dbus"
)

// DBusCmd возвращает команду для управления D-Bus сервисом
func DBusCmd() *cli.Command {
	return &cli.Command{
		Name:  "dbus",
		Usage: gotext.Get("Manage D-Bus service"),
		Subcommands: []*cli.Command{
			DBusStatusCmd(),
			DBusStartCmd(),
			DBusStopCmd(),
			DBusSearchCmd(),
			DBusInstallCmd(),
			DBusRemoveCmd(),
		},
	}
}

// DBusStatusCmd возвращает статус D-Bus сервиса
func DBusStatusCmd() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: gotext.Get("Show D-Bus service status"),
		Action: func(c *cli.Context) error {
			conn, err := dbus.ConnectSessionBus()
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Failed to connect to D-Bus"), err)
			}
			defer conn.Close()

			// Проверяем доступность сервиса
			obj := conn.Object(alrdbus.DBusWellKnownName, alrdbus.DBusObjectPath)
			var version string
			err = obj.Call(alrdbus.ManagerInterfaceName+".GetVersion", 0).Store(&version)
			if err != nil {
				fmt.Println(gotext.Get("D-Bus service: not running"))
				return nil
			}

			fmt.Println(gotext.Get("D-Bus service: running"))
			fmt.Printf("Version: %s\n", version)

			return nil
		},
	}
}

// DBusStartCmd запускает D-Bus сервис
func DBusStartCmd() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: gotext.Get("Start D-Bus service"),
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "session",
				Usage: "Use session bus (default)",
				Value: true,
			},
		},
		Action: func(c *cli.Context) error {
			// Запускаем как отдельный процесс
			cmd := exec.Command("alr", "_internal-dbus-service")
			if c.Bool("session") {
				cmd.Args = append(cmd.Args, "--session")
			}

			if err := cmd.Start(); err != nil {
				return cliutils.FormatCliExit(gotext.Get("Failed to start D-Bus service"), err)
			}

			fmt.Println(gotext.Get("D-Bus service started"))
			return nil
		},
	}
}

// DBusStopCmd останавливает D-Bus сервис
func DBusStopCmd() *cli.Command {
	return &cli.Command{
		Name:  "stop",
		Usage: gotext.Get("Stop D-Bus service"),
		Action: func(c *cli.Context) error {
			// Для остановки нужно убить процесс
			// Ищем процесс по имени
			cmd := exec.Command("pkill", "-f", "alr _internal-dbus-service")
			if err := cmd.Run(); err != nil {
				return cliutils.FormatCliExit(gotext.Get("Failed to stop D-Bus service"), err)
			}

			fmt.Println(gotext.Get("D-Bus service stopped"))
			return nil
		},
	}
}

// DBusSearchCmd ищет пакеты через D-Bus
func DBusSearchCmd() *cli.Command {
	return &cli.Command{
		Name:    "search",
		Usage:   gotext.Get("Search packages via D-Bus"),
		Aliases: []string{"s"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
				Usage:   gotext.Get("Search by name"),
			},
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   gotext.Get("Search by description"),
			},
			&cli.StringFlag{
				Name:    "repository",
				Aliases: []string{"r"},
				Usage:   gotext.Get("Search by repository"),
			},
		},
		Action: func(c *cli.Context) error {
			conn, err := dbus.ConnectSessionBus()
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Failed to connect to D-Bus"), err)
			}
			defer conn.Close()

			// Формируем фильтры
			filters := make(map[string]dbus.Variant)
			if c.IsSet("name") {
				filters["name"] = dbus.MakeVariant(c.String("name"))
			}
			if c.IsSet("description") {
				filters["description"] = dbus.MakeVariant(c.String("description"))
			}
			if c.IsSet("repository") {
				filters["repository"] = dbus.MakeVariant(c.String("repository"))
			}

			// Запрос к D-Bus
			obj := conn.Object(alrdbus.DBusWellKnownName, alrdbus.DBusObjectPath)

			query := c.Args().First()
			if query == "" {
				query = c.String("name")
			}

			var results []alrdbus.PackageInfo
			err = obj.Call(alrdbus.ManagerInterfaceName+".SearchPackages", 0, query, filters).Store(&results)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Search failed"), err)
			}

			// Вывод результатов
			if len(results) == 0 {
				fmt.Println(gotext.Get("No packages found"))
				return nil
			}

			fmt.Printf("Found %d packages:\n", len(results))
			for _, pkg := range results {
				installed := ""
				if pkg.Installed {
					installed = " [installed]"
				}
				fmt.Printf("  %s/%s %s%s\n", pkg.Repository, pkg.Name, pkg.Version, installed)
				if pkg.Summary != "" {
					fmt.Printf("    %s\n", pkg.Summary)
				}
			}

			return nil
		},
	}
}

// DBusInstallCmd устанавливает пакет через D-Bus
func DBusInstallCmd() *cli.Command {
	return &cli.Command{
		Name:    "install",
		Usage:   gotext.Get("Install package via D-Bus"),
		Aliases: []string{"in"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "clean",
				Aliases: []string{"c"},
				Usage:   gotext.Get("Build from scratch"),
			},
			&cli.BoolFlag{
				Name:  "no-progress",
				Usage: gotext.Get("Don't show progress"),
			},
		},
		Action: func(c *cli.Context) error {
			if c.Args().Len() < 1 {
				return cliutils.FormatCliExit(gotext.Get("Package name required"), nil)
			}

			pkgArg := c.Args().First()

			// Парсим repo/name
			var repo, name string
			if strings.Contains(pkgArg, "/") {
				parts := strings.SplitN(pkgArg, "/", 2)
				repo = parts[0]
				name = parts[1]
			} else {
				name = pkgArg
				repo = "" // Будет использоваться дефолтный
			}

			conn, err := dbus.ConnectSessionBus()
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Failed to connect to D-Bus"), err)
			}
			defer conn.Close()

			obj := conn.Object(alrdbus.DBusWellKnownName, alrdbus.DBusObjectPath)

			// Получаем пакет
			var pkgPath dbus.ObjectPath
			err = obj.Call(alrdbus.ManagerInterfaceName+".GetPackage", 0, name, repo).Store(&pkgPath)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Failed to get package"), err)
			}

			// Опции установки
			opts := make(map[string]dbus.Variant)
			opts["clean"] = dbus.MakeVariant(c.Bool("clean"))
			opts["interactive"] = dbus.MakeVariant(c.Bool("interactive"))

			// Устанавливаем
			pkgObj := conn.Object(alrdbus.DBusWellKnownName, pkgPath)
			var jobPath dbus.ObjectPath
			err = pkgObj.Call(alrdbus.PackageInterfaceName+".Install", 0, opts).Store(&jobPath)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Failed to start installation"), err)
			}

			fmt.Printf("Installation started: %s\n", jobPath)

			// Если не --no-progress, следим за прогрессом
			if !c.Bool("no-progress") {
				if err := watchJob(conn, jobPath); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

// DBusRemoveCmd удаляет пакет через D-Bus
func DBusRemoveCmd() *cli.Command {
	return &cli.Command{
		Name:    "remove",
		Usage:   gotext.Get("Remove package via D-Bus"),
		Aliases: []string{"rm"},
		Action: func(c *cli.Context) error {
			if c.Args().Len() < 1 {
				return cliutils.FormatCliExit(gotext.Get("Package name required"), nil)
			}

			pkgArg := c.Args().First()

			// Парсим repo/name
			var repo, name string
			if strings.Contains(pkgArg, "/") {
				parts := strings.SplitN(pkgArg, "/", 2)
				repo = parts[0]
				name = parts[1]
			} else {
				name = pkgArg
				repo = ""
			}

			conn, err := dbus.ConnectSessionBus()
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Failed to connect to D-Bus"), err)
			}
			defer conn.Close()

			obj := conn.Object(alrdbus.DBusWellKnownName, alrdbus.DBusObjectPath)

			// Получаем пакет
			var pkgPath dbus.ObjectPath
			err = obj.Call(alrdbus.ManagerInterfaceName+".GetPackage", 0, name, repo).Store(&pkgPath)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Failed to get package"), err)
			}

			// Удаляем
			opts := make(map[string]dbus.Variant)
			opts["interactive"] = dbus.MakeVariant(c.Bool("interactive"))

			pkgObj := conn.Object(alrdbus.DBusWellKnownName, pkgPath)
			var jobPath dbus.ObjectPath
			err = pkgObj.Call(alrdbus.PackageInterfaceName+".Remove", 0, opts).Store(&jobPath)
			if err != nil {
				return cliutils.FormatCliExit(gotext.Get("Failed to start removal"), err)
			}

			fmt.Printf("Removal started: %s\n", jobPath)

			// Следим за прогрессом
			if !c.Bool("no-progress") {
				if err := watchJob(conn, jobPath); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

// watchJob следит за выполнением задачи
func watchJob(conn *dbus.Conn, jobPath dbus.ObjectPath) error {
	jobObj := conn.Object(alrdbus.DBusWellKnownName, jobPath)

	// Получаем информацию о задаче
	var info alrdbus.JobInfo
	err := jobObj.Call(alrdbus.JobInterfaceName+".GetInfo", 0).Store(&info)
	if err != nil {
		return cliutils.FormatCliExit(gotext.Get("Failed to get job info"), err)
	}

	slog.Info("Job started", "type", info.Type, "package", info.PackageName)

	// Ждем завершения
	var success bool
	var errMsg string
	err = jobObj.Call(alrdbus.JobInterfaceName+".Wait", 0, int32(0)).Store(&success, &errMsg)
	if err != nil {
		return cliutils.FormatCliExit(gotext.Get("Failed to wait for job"), err)
	}

	if success {
		fmt.Println(gotext.Get("Operation completed successfully"))
	} else {
		return cliutils.FormatCliExit(gotext.Get("Operation failed"), fmt.Errorf("%s", errMsg))
	}

	return nil
}
