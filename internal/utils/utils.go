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

// EnsureTempDirWithRootOwner создает каталог в /tmp/alr с правами для группы wheel
// Все каталоги в /tmp/alr принадлежат root:wheel с правами 775
// Для других каталогов использует стандартные права
func EnsureTempDirWithRootOwner(path string, mode os.FileMode) error {
	if strings.HasPrefix(path, "/tmp/alr") {
		// Сначала создаем директорию обычным способом
		err := os.MkdirAll(path, mode)
		if err != nil {
			return err
		}
		
		// Все каталоги в /tmp/alr доступны для группы wheel
		// Устанавливаем setgid бит (2775), чтобы новые файлы наследовали группу
		permissions := "2775"
		group := "wheel"
		
		// Устанавливаем права с setgid битом
		err = exec.Command("sudo", "chmod", permissions, path).Run()
		if err != nil {
			return err
		}
		
		// Устанавливаем владельца root:wheel
		return exec.Command("sudo", "chown", "root:"+group, path).Run()
	}
	
	// Для остальных каталогов обычное создание
	return os.MkdirAll(path, mode)
}
