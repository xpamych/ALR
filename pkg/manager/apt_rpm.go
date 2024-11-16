/*
 * ALR - Any Linux Repository
 * Copyright (C) 2024 Евгений Храмов
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package manager

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// APTRpm represents the APT-RPM package manager
type APTRpm struct {
	rootCmd string
}

func (*APTRpm) Exists() bool {
	cmd := exec.Command("apt-config", "dump")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "RPM")
}

func (*APTRpm) Name() string {
	return "apt-rpm"
}

func (*APTRpm) Format() string {
	return "rpm"
}

func (a *APTRpm) SetRootCmd(s string) {
	a.rootCmd = s
}

func (a *APTRpm) Sync(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apt-get", "update")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt-get: sync: %w", err)
	}
	return nil
}

func (a *APTRpm) Install(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apt-get", "install")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt-get: install: %w", err)
	}
	return nil
}

func (a *APTRpm) InstallLocal(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	return a.Install(opts, pkgs...)
}

func (a *APTRpm) Remove(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apt-get", "remove")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt-get: remove: %w", err)
	}
	return nil
}

func (a *APTRpm) Upgrade(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	return a.Install(opts, pkgs...)
}

func (a *APTRpm) UpgradeAll(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := a.getCmd(opts, "apt-get", "dist-upgrade")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt-get: upgradeall: %w", err)
	}
	return nil
}
func (y *APTRpm) ListInstalled(opts *Opts) (map[string]string, error) {
	out := map[string]string{}
	cmd := exec.Command("rpm", "-qa", "--queryformat", "%{NAME}\u200b%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\\n")

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
		version = strings.TrimPrefix(version, "0:")
		out[name] = version
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (a *APTRpm) getCmd(opts *Opts, mgrCmd string, args ...string) *exec.Cmd {
	var cmd *exec.Cmd
	if opts.AsRoot {
		cmd = exec.Command(getRootCmd(a.rootCmd), mgrCmd)
		cmd.Args = append(cmd.Args, opts.Args...)
		cmd.Args = append(cmd.Args, args...)
	} else {
		cmd = exec.Command(mgrCmd, args...)
	}

	if opts.NoConfirm {
		cmd.Args = append(cmd.Args, "-y")
	}

	return cmd
}
