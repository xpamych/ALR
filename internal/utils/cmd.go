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

package utils

import (
	"os"
	"os/exec"
	"os/user"

	"github.com/leonelquinteros/gotext"
	"github.com/urfave/cli/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	appbuilder "gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils/app_builder"
)

// IsNotRoot проверяет, что текущий пользователь не является root
func IsNotRoot() bool {
	return os.Getuid() != 0
}

// EnuseIsPrivilegedGroupMember проверяет, что пользователь является членом привилегированной группы (wheel)
func EnuseIsPrivilegedGroupMember() error {
	// В CI пропускаем проверку группы wheel
	if os.Getenv("CI") == "true" {
		return nil
	}
	
	// Если пользователь root, пропускаем проверку
	if os.Geteuid() == 0 {
		return nil
	}
	
	currentUser, err := user.Current()
	if err != nil {
		return err
	}

	privilegedGroup := GetPrivilegedGroup()
	group, err := user.LookupGroup(privilegedGroup)
	if err != nil {
		return err
	}

	groups, err := currentUser.GroupIds()
	if err != nil {
		return err
	}

	for _, gid := range groups {
		if gid == group.Gid {
			return nil
		}
	}
	return cliutils.FormatCliExit(gotext.Get("You need to be a %s member to perform this action", privilegedGroup), nil)
}

func RootNeededAction(f cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		deps, err := appbuilder.
			New(ctx.Context).
			WithConfig().
			Build()
		if err != nil {
			return err
		}
		defer deps.Defer()

		if IsNotRoot() {
			if !deps.Cfg.UseRootCmd() {
				return cli.Exit(gotext.Get("You need to be root to perform this action"), 1)
			}
			executable, err := os.Executable()
			if err != nil {
				return cliutils.FormatCliExit("failed to get executable path", err)
			}
			args := append([]string{executable}, os.Args[1:]...)
			cmd := exec.Command(deps.Cfg.RootCmd(), args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
		return f(ctx)
	}
}
