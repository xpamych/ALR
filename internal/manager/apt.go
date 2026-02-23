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

	resolvedPkgs := make([]string, 0, len(pkgs))
	for _, pkg := range pkgs {
		resolved := a.resolvePackageName(pkg)
		resolvedPkgs = append(resolvedPkgs, resolved)
	}

	cmd := a.getCmd(opts, "apt", "remove")
	cmd.Args = append(cmd.Args, resolvedPkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("apt: remove: %w", err)
	}
	return nil
}

func (a *APT) resolvePackageName(pkg string) string {
	cmd := exec.Command("dpkg-query", "-f", "${Status}", "-W", pkg)
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), "install ok installed") {
		return pkg
	}

	cmd = exec.Command("dpkg-query", "-W", "-f", "${Package}\t${Provides}\n")
	output, err = cmd.Output()
	if err != nil {
		return pkg
	}

	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		pkgName := parts[0]
		provides := parts[1]

		for _, p := range strings.Split(provides, ", ") {
			p = strings.TrimSpace(p)
			provName := strings.Split(p, " ")[0]
			if provName == pkg {
				return pkgName
			}
		}
	}

	return pkg
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
	resolved := a.resolvePackageName(pkg)
	cmd := exec.Command("dpkg-query", "-f", "${Status}", "-W", resolved)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return false, nil
			}
		}
		return false, fmt.Errorf("apt: isinstalled: %w, output: %s", err, output)
	}

	status := strings.TrimSpace(string(output))
	return strings.Contains(status, "install ok installed"), nil
}

func (a *APT) ListAvailable(prefix string) ([]string, error) {
	return aptCacheListAvailable(prefix)
}

func (a *APT) GetInstalledVersion(pkg string) (string, error) {
	resolved := a.resolvePackageName(pkg)
	cmd := exec.Command("dpkg-query", "-f", "${Version}", "-W", resolved)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return "", nil
			}
		}
		return "", fmt.Errorf("apt: getinstalledversion: %w, output: %s", err, output)
	}

	return strings.TrimSpace(string(output)), nil
}
