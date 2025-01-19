/*
 * ALR - Any Linux Repository
 * ALR - Любой Linux Репозиторий
 * Copyright (C) 2024 Евгений Храмов
 *
 * This program является свободным: вы можете распространять его и/или изменять
 * на условиях GNU General Public License, опубликованной Free Software Foundation,
 * либо версии 3 лицензии, либо (по вашему выбору) любой более поздней версии.
 *
 * Это программное обеспечение распространяется в надежде, что оно будет полезным,
 * но БЕЗ КАКИХ-ЛИБО ГАРАНТИЙ; без подразумеваемой гарантии
 * КОММЕРЧЕСКОЙ ПРИГОДНОСТИ или ПРИГОДНОСТИ ДЛЯ ОПРЕДЕЛЕННОЙ ЦЕЛИ.
 * Подробности см. в GNU General Public License.
 *
 * Вы должны были получить копию GNU General Public License
 * вместе с этой программой. Если нет, см. <http://www.gnu.org/licenses/>.
 */

package manager

import (
	"fmt"
	"os/exec"
)

// DNF представляет менеджер пакетов DNF
type DNF struct {
	CommonRPM
	rootCmd string // rootCmd хранит команду, используемую для выполнения команд с правами root
}

// Exists проверяет, доступен ли DNF в системе, возвращает true если да
func (*DNF) Exists() bool {
	_, err := exec.LookPath("dnf")
	return err == nil
}

// Name возвращает имя менеджера пакетов, в данном случае "dnf"
func (*DNF) Name() string {
	return "dnf"
}

// Format возвращает формат пакетов "rpm", используемый DNF
func (*DNF) Format() string {
	return "rpm"
}

// SetRootCmd устанавливает команду, используемую для выполнения операций с правами root
func (d *DNF) SetRootCmd(s string) {
	d.rootCmd = s
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

// getCmd создает и возвращает команду exec.Cmd для менеджера пакетов DNF
func (d *DNF) getCmd(opts *Opts, mgrCmd string, args ...string) *exec.Cmd {
	var cmd *exec.Cmd
	if opts.AsRoot {
		cmd = exec.Command(getRootCmd(d.rootCmd), mgrCmd)
		cmd.Args = append(cmd.Args, opts.Args...)
		cmd.Args = append(cmd.Args, args...)
	} else {
		cmd = exec.Command(mgrCmd, args...)
	}

	if opts.NoConfirm {
		cmd.Args = append(cmd.Args, "-y") // Добавляет параметр автоматического подтверждения (-y)
	}

	return cmd
}
