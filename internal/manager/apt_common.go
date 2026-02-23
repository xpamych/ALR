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
	"bufio"
	"fmt"
	"os/exec"
)

func aptCacheListAvailable(prefix string) ([]string, error) {
	cmd := exec.Command("apt-cache", "pkgnames", prefix)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("apt-cache: listavailable: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("apt-cache: listavailable: %w", err)
	}

	var pkgs []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		pkgs = append(pkgs, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("apt-cache: listavailable: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("apt-cache: listavailable: %w", err)
	}

	return pkgs, nil
}
