// This file was originally part of the project "LURE - Linux User REpository", created by Elara Musayelyan.
// It has been modified as part of "ALR - Any Linux Repository" by Евгений Храмов.
//
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
	"path/filepath"

	"plemya-x.ru/alr/internal/config"
	"plemya-x.ru/alr/internal/db"
	"plemya-x.ru/alr/internal/types"
	"plemya-x.ru/alr/pkg/loggerctx"
)

// InstallPkgs устанавливает нативные пакеты с использованием менеджера пакетов,
// затем строит и устанавливает пакеты ALR
func InstallPkgs(ctx context.Context, alrPkgs []db.Package, nativePkgs []string, opts types.BuildOpts) {
	log := loggerctx.From(ctx) // Инициализируем логгер из контекста

	if len(nativePkgs) > 0 {
		err := opts.Manager.Install(nil, nativePkgs...)
		// Если есть нативные пакеты, выполняем их установку
		if err != nil {
			log.Fatal("Error installing native packages").Err(err).Send()
			// Логируем и завершаем выполнение при ошибке
		}
	}

	InstallScripts(ctx, GetScriptPaths(ctx, alrPkgs), opts)
	// Устанавливаем скрипты сборки через функцию InstallScripts
}

// GetScriptPaths возвращает срез путей к скриптам, соответствующий
// данным пакетам
func GetScriptPaths(ctx context.Context, pkgs []db.Package) []string {
	var scripts []string
	for _, pkg := range pkgs {
		// Для каждого пакета создаем путь к скрипту сборки
		scriptPath := filepath.Join(config.GetPaths(ctx).RepoDir, pkg.Repository, pkg.Name, "alr.sh")
		scripts = append(scripts, scriptPath)
	}
	return scripts
}

// InstallScripts строит и устанавливает переданные alr скрипты сборки
func InstallScripts(ctx context.Context, scripts []string, opts types.BuildOpts) {
	log := loggerctx.From(ctx) // Получаем логгер из контекста
	for _, script := range scripts {
		opts.Script = script // Устанавливаем текущий скрипт в опции
		builtPkgs, _, err := BuildPackage(ctx, opts)
		// Выполняем сборку пакета
		if err != nil {
			log.Fatal("Error building package").Err(err).Send()
			// Логируем и завершаем выполнение при ошибке сборки
		}

		err = opts.Manager.InstallLocal(nil, builtPkgs...)
		// Устанавливаем локально собранные пакеты
		if err != nil {
			log.Fatal("Error installing package").Err(err).Send()
			// Логируем и завершаем выполнение при ошибке установки
		}
	}
}
