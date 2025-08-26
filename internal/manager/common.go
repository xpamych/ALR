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

package manager

import (
	"os"
	"os/exec"
)

type CommonPackageManager struct {
	noConfirmArg string
}

func (m *CommonPackageManager) getCmd(opts *Opts, mgrCmd string, args ...string) *exec.Cmd {
	var cmd *exec.Cmd
	
	// Проверяем, нужно ли повышение привилегий
	isRoot := os.Geteuid() == 0
	isCI := os.Getenv("CI") == "true"
	
	if !isRoot && !isCI {
		// Если не root и не в CI, используем sudo
		cmd = exec.Command("sudo", mgrCmd)
	} else {
		// Если root или в CI, запускаем напрямую
		cmd = exec.Command(mgrCmd)
	}
	
	cmd.Args = append(cmd.Args, opts.Args...)
	cmd.Args = append(cmd.Args, args...)

	if opts.NoConfirm {
		cmd.Args = append(cmd.Args, m.noConfirmArg)
	}

	return cmd
}
