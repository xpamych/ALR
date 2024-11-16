/*
 * ALR - Any Linux Repository
 * ALR - Любой Linux Репозиторий
 * Copyright (C) 2024 Евгений Храмов
 *
 * This program is free software: you can redistribute it and/or modify
 * Это программное обеспечение является свободным: вы можете распространять его и/или изменять
 * it under the terms of the GNU General Public License as published by
 * на условиях GNU General Public License, опубликованной
 * the Free Software Foundation, either version 3 of the License, or
 * Free Software Foundation, либо версии 3 лицензии, либо
 * (at your option) any later version.
 * (по вашему усмотрению) любой более поздней версии.
 *
 * This program is distributed in the hope that it will be useful,
 * Это программное обеспечение распространяется в надежде, что оно будет полезным,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * но БЕЗ КАКОЙ-ЛИБО ГАРАНТИИ; даже без подразумеваемой гарантии
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * КОММЕРЧЕСКОЙ ПРИГОДНОСТИ или ПРИГОДНОСТИ ДЛЯ ОПРЕДЕЛЕННОЙ ЦЕЛИ.
 * GNU General Public License for more details.
 * Подробности смотрите в GNU General Public License.
 *
 * You should have received a copy of the GNU General Public License
 * Вы должны были получить копию GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 * вместе с этой программой. Если нет, посмотрите <http://www.gnu.org/licenses/>.
 */

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
