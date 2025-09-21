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
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"golang.org/x/sys/unix"
)

func NoNewPrivs() error {
	return unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0)
}

// EnsureTempDirWithRootOwner создает каталог в /tmp/alr или /var/cache/alr с правами для привилегированной группы
// Все каталоги в /tmp/alr и /var/cache/alr принадлежат root:привилегированная_группа с правами 2775
// Для других каталогов использует стандартные права
func EnsureTempDirWithRootOwner(path string, mode os.FileMode) error {
	needsElevation := strings.HasPrefix(path, "/tmp/alr") || strings.HasPrefix(path, "/var/cache/alr")

	if needsElevation {
		// В CI или если мы уже root, не нужно использовать sudo
		isRoot := os.Geteuid() == 0
		isCI := os.Getenv("CI") == "true"

		// В CI создаем директории с обычными правами
		if isCI {
			err := os.MkdirAll(path, mode)
			if err != nil {
				return err
			}
			// В CI не используем группу wheel и не меняем права
			// Устанавливаем базовые права 777 для временных каталогов
			chmodCmd := exec.Command("chmod", "777", path)
			chmodCmd.Run() // Игнорируем ошибки
			return nil
		}

		// Для обычной работы устанавливаем права и привилегированную группу
		permissions := "2775"
		group := GetPrivilegedGroup()

		var mkdirCmd, chmodCmd, chownCmd *exec.Cmd
		if isRoot {
			// Выполняем команды напрямую без sudo
			mkdirCmd = exec.Command("mkdir", "-p", path)
			chmodCmd = exec.Command("chmod", permissions, path)
			chownCmd = exec.Command("chown", "root:"+group, path)
		} else {
			// Используем sudo для всех операций с привилегированными каталогами
			mkdirCmd = exec.Command("sudo", "mkdir", "-p", path)
			chmodCmd = exec.Command("sudo", "chmod", permissions, path)
			chownCmd = exec.Command("sudo", "chown", "root:"+group, path)
		}

		// Создаем директорию через sudo если нужно
		err := mkdirCmd.Run()
		if err != nil {
			// Игнорируем ошибку если директория уже существует
			if !isRoot {
				// Проверяем существует ли директория
				if _, statErr := os.Stat(path); statErr != nil {
					return fmt.Errorf("не удалось создать директорию %s: %w", path, err)
				}
			}
		}

		// Устанавливаем права с setgid битом для наследования группы
		err = chmodCmd.Run()
		if err != nil {
			if !isRoot {
				return fmt.Errorf("не удалось установить права на %s: %w", path, err)
			}
		}

		// Устанавливаем владельца root:группа
		err = chownCmd.Run()
		if err != nil {
			if !isRoot {
				return fmt.Errorf("не удалось установить владельца на %s: %w", path, err)
			}
		}

		return nil
	}

	// Для остальных каталогов обычное создание
	return os.MkdirAll(path, mode)
}

// IsUserInGroup проверяет, состоит ли пользователь в указанной группе
func IsUserInGroup(username, groupname string) bool {
	u, err := user.Lookup(username)
	if err != nil {
		return false
	}

	groups, err := u.GroupIds()
	if err != nil {
		return false
	}

	targetGroup, err := user.LookupGroup(groupname)
	if err != nil {
		return false
	}

	for _, gid := range groups {
		if gid == targetGroup.Gid {
			return true
		}
	}
	return false
}

// CheckUserPrivileges проверяет, что пользователь имеет необходимые привилегии для работы с ALR
// Пользователь должен быть root или состоять в группе wheel/sudo
func CheckUserPrivileges() error {
	// Если пользователь root - все в порядке
	if os.Geteuid() == 0 {
		return nil
	}

	// В CI не проверяем привилегии
	if os.Getenv("CI") == "true" {
		return nil
	}

	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("не удалось получить информацию о текущем пользователе: %w", err)
	}

	privilegedGroup := GetPrivilegedGroup()

	// Проверяем членство в привилегированной группе
	if !IsUserInGroup(currentUser.Username, privilegedGroup) {
		return fmt.Errorf("пользователь %s не имеет необходимых привилегий для работы с ALR.\n"+
			"Для работы с ALR необходимо быть пользователем root или состоять в группе %s.\n"+
			"Выполните команду: sudo usermod -a -G %s %s\n"+
			"Затем перезайдите в систему или выполните: newgrp %s",
			currentUser.Username, privilegedGroup, privilegedGroup, currentUser.Username, privilegedGroup)
	}

	return nil
}
