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
	"strings"

	"golang.org/x/sys/unix"
)

func NoNewPrivs() error {
	return unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0)
}

// EnsureTempDirWithRootOwner создает каталог в /tmp/alr с правами для привилегированной группы
// Все каталоги в /tmp/alr принадлежат root:привилегированная_группа с правами 775
// Для других каталогов использует стандартные права
func EnsureTempDirWithRootOwner(path string, mode os.FileMode) error {
	if strings.HasPrefix(path, "/tmp/alr") {
		// Сначала создаем директорию обычным способом
		err := os.MkdirAll(path, mode)
		if err != nil {
			return err
		}
		
		// В CI или если мы уже root, не нужно использовать sudo
		isRoot := os.Geteuid() == 0
		isCI := os.Getenv("CI") == "true"
		
		// В CI создаем директории с обычными правами
		if isCI {
			// В CI не используем группу wheel и не меняем права
			// Устанавливаем базовые права 777 для временных каталогов
			chmodCmd := exec.Command("chmod", "777", path)
			chmodCmd.Run() // Игнорируем ошибки
			return nil
		}
		
		// Для обычной работы устанавливаем права и привилегированную группу
		permissions := "2775"
		group := GetPrivilegedGroup()
		
		var chmodCmd, chownCmd *exec.Cmd
		if isRoot {
			// Выполняем команды напрямую без sudo
			chmodCmd = exec.Command("chmod", permissions, path)
			chownCmd = exec.Command("chown", "root:"+group, path)
		} else {
			// Используем sudo для обычных пользователей
			chmodCmd = exec.Command("sudo", "chmod", permissions, path)
			chownCmd = exec.Command("sudo", "chown", "root:"+group, path)
		}
		
		// Устанавливаем права с setgid битом
		err = chmodCmd.Run()
		if err != nil {
			// Для root игнорируем ошибки, если группа не существует
			if !isRoot {
				return err
			}
		}
		
		// Устанавливаем владельца root:wheel
		err = chownCmd.Run()
		if err != nil {
			// Для root игнорируем ошибки, если группа не существует
			if !isRoot {
				return err
			}
		}
		
		return nil
	}
	
	// Для остальных каталогов обычное создание
	return os.MkdirAll(path, mode)
}
