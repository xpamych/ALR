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
)

type DNF struct {
	CommonPackageManager
	CommonRPM
}

func NewDNF() *DNF {
	return &DNF{
		CommonPackageManager: CommonPackageManager{
			noConfirmArg: "-y",
		},
	}
}

func (*DNF) Exists() bool {
	_, err := exec.LookPath("dnf")
	return err == nil
}

func (*DNF) Name() string {
	return "dnf"
}

func (*DNF) Format() string {
	return "rpm"
}

// Sync выполняет upgrade всех установленных пакетов, обновляя их до более новых версий
func (d *DNF) Sync(opts *Opts) error {
	opts = ensureOpts(opts) // Гарантирует, что opts не равен nil и содержит допустимые значения
	cmd := d.getCmd(opts, "dnf", "upgrade")
	setCmdEnv(cmd)   // Устанавливает переменные окружения для команды
	err := cmd.Run() // Выполняет команду
	if err != nil {
		return fmt.Errorf("dnf: sync: %w", err)
	}
	return nil
}

// Install устанавливает указанные пакеты с помощью DNF
func (d *DNF) Install(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := d.getCmd(opts, "dnf", "install", "--allowerasing")
	cmd.Args = append(cmd.Args, pkgs...) // Добавляем названия пакетов к команде
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("dnf: install: %w", err)
	}
	return nil
}

// InstallLocal расширяет метод Install для установки пакетов, расположенных локально
func (d *DNF) InstallLocal(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	return d.Install(opts, pkgs...)
}

// Remove удаляет указанные пакеты с помощью DNF
func (d *DNF) Remove(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := d.getCmd(opts, "dnf", "remove")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("dnf: remove: %w", err)
	}
	return nil
}

// Upgrade обновляет указанные пакеты до более новых версий
func (d *DNF) Upgrade(opts *Opts, pkgs ...string) error {
	opts = ensureOpts(opts)
	cmd := d.getCmd(opts, "dnf", "upgrade")
	cmd.Args = append(cmd.Args, pkgs...)
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("dnf: upgrade: %w", err)
	}
	return nil
}

func (d *DNF) ListAvailable(prefix string) ([]string, error) {
	cmd := exec.Command("dnf", "repoquery", "--qf", "%{name}\n", "--quiet", prefix+"*")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("dnf: listavailable: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("dnf: listavailable: %w", err)
	}

	seen := make(map[string]struct{})
	var pkgs []string
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		name := scanner.Text()
		if name == "" {
			continue
		}
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			pkgs = append(pkgs, name)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("dnf: listavailable: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("dnf: listavailable: %w", err)
	}

	return pkgs, nil
}

// UpgradeAll обновляет все установленные пакеты
func (d *DNF) UpgradeAll(opts *Opts) error {
	opts = ensureOpts(opts)
	cmd := d.getCmd(opts, "dnf", "upgrade")
	setCmdEnv(cmd)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("dnf: upgradeall: %w", err)
	}
	return nil
}
