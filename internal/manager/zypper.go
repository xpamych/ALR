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

// Zypper represents the Zypper package manager
type Zypper struct {
	CommonPackageManager
	CommonRPM
}

func NewZypper() *Zypper {
	return &Zypper{
		CommonPackageManager: CommonPackageManager{
			noConfirmArg: "-y",
		},
	}
}

func (*Zypper) Exists() bool {
	_, err := exec.LookPath("zypper")
	return err == nil
}

func (*Zypper) Name() string {
	return "zypper"
}

func (*Zypper) Format() string {
	return "rpm"
}

func (z *Zypper) Sync(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := z.getCmd(opts, "zypper", "refresh")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: sync: %w", err)
	}
	return nil
}

func (z *Zypper) Install(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := z.getCmd(opts, "zypper", "install", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: install: %w", err)
	}
	return nil
}

func (z *Zypper) InstallLocal(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	return z.Install(opts, pkgs...)
}

func (z *Zypper) Remove(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := z.getCmd(opts, "zypper", "remove", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: remove: %w", err)
	}
	return nil
}

func (z *Zypper) Upgrade(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := z.getCmd(opts, "zypper", "update", "-y")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: upgrade: %w", err)
	}
	return nil
}

func (z *Zypper) ListAvailable(prefix string) ([]string, error) {
	cmd := exec.Command("zypper", "--quiet", "search", "--type", "package", prefix+"*")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("zypper: listavailable: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("zypper: listavailable: %w", err)
	}

	seen := make(map[string]struct{})
	var pkgs []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		// zypper table format: "S | Name | Summary | Type"
		// Skip separator lines and headers
		if !strings.Contains(line, "|") {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimSpace(fields[1])
		if name == "" || name == "Name" {
			continue
		}
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			pkgs = append(pkgs, name)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("zypper: listavailable: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 104 {
			return nil, nil
		}
		return nil, fmt.Errorf("zypper: listavailable: %w", err)
	}

	return pkgs, nil
}

func (z *Zypper) UpgradeAll(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := z.getCmd(opts, "zypper", "update", "-y")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("zypper: upgradeall: %w", err)
	}
	return nil
}
