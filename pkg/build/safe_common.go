// ALR - Any Linux Repository
// Copyright (C) 2025 Евгений Храмов
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

package build

import (
	"os"
	"os/exec"
	"strings"
)

func setCommonCmdEnv(cmd *exec.Cmd) {
	cmd.Env = []string{
		"HOME=/var/cache/alr",
		"LOGNAME=alr",
		"USER=alr",
		"PATH=/usr/bin:/bin:/usr/local/bin",
	}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "LANG=") ||
			strings.HasPrefix(env, "LANGUAGE=") ||
			strings.HasPrefix(env, "LC_") ||
			strings.HasPrefix(env, "ALR_LOG_LEVEL=") {
			cmd.Env = append(cmd.Env, env)
		}
	}
}
