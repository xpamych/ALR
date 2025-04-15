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
	"context"
	"log/slog"

	"github.com/leonelquinteros/gotext"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cpu"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/manager"
)

type Checker struct {
	mgr manager.Manager
}

func (c *Checker) PerformChecks(
	ctx context.Context,
	input *BuildInput,
	vars *types.BuildVars,
) (bool, error) {
	if !cpu.IsCompatibleWith(cpu.Arch(), vars.Architectures) { // Проверяем совместимость архитектуры
		cont, err := cliutils.YesNoPrompt(
			ctx,
			gotext.Get("Your system's CPU architecture doesn't match this package. Do you want to build anyway?"),
			input.opts.Interactive,
			true,
		)
		if err != nil {
			return false, err
		}

		if !cont {
			return false, nil
		}
	}

	installed, err := c.mgr.ListInstalled(nil)
	if err != nil {
		return false, err
	}

	filename, err := pkgFileName(input, vars)
	if err != nil {
		return false, err
	}

	if instVer, ok := installed[filename]; ok { // Если пакет уже установлен, выводим предупреждение
		slog.Warn(gotext.Get("This package is already installed"),
			"name", vars.Name,
			"version", instVer,
		)
	}

	return true, nil
}
