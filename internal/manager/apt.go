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

package manager

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// APT represents the APT package manager
type APT struct {
	CommonPackageManager
}

func NewAPT() *APT {
	return &APT{
		CommonPackageManager: CommonPackageManager{
			noConfirmArg: "-y",
		},
	}
}

func (*APT) Exists() bool {
	_, err := exec.LookPath("apt")
	return err == nil
}

func (*APT) Name() string {
	return "apt"
}

func (*APT) Format() string {
	return "deb"
}

func (a *APT) Sync(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apt", "update")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt: sync: %w", err)
	}
	return nil
}

func (a *APT) Install(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apt", "install")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt: install: %w", err)
	}
	return nil
}

func (a *APT) InstallLocal(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	return a.Install(opts, pkgs...)
}

func (a *APT) Remove(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apt", "remove")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt: remove: %w", err)
	}
	return nil
}

func (a *APT) Upgrade(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	return a.Install(opts, pkgs...)
}

func (a *APT) UpgradeAll(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apt", "upgrade")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt: upgradeall: %w", err)
	}
	return nil
}

func (a *APT) ListInstalled(opts *Opts) (map[string]string, error) {
	out := map[string]string{}
	cmd := exec.Command("dpkg-query", "-f", "${Package}\u200b${Version}\\n", "-W")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		name, version, ok := strings.Cut(scanner.Text(), "\u200b")
		if !ok {
			continue
		}
		out[name] = version
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (a *APT) IsInstalled(pkg string) (bool, error) {
	cmd := exec.Command("dpkg-query", "-f", "${Status}", "-W", pkg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Код выхода 1 означает что пакет не найден
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, fmt.Errorf("apt: isinstalled: %w, output: %s", err, output)
	}

	status := strings.TrimSpace(string(output))
	// Проверяем что пакет действительно установлен (статус должен содержать "install ok installed")
	return strings.Contains(status, "install ok installed"), nil
}
