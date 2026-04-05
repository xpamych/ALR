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

package build

import (
	"context"
	"fmt"
	"log/slog"

	"git.alr-pkg.ru/Plemya-x/ALR/internal/overrides"
	"git.alr-pkg.ru/Plemya-x/ALR/pkg/alrsh"
)

// DependencyNode представляет узел в дереве зависимостей
type DependencyNode struct {
	Package      *alrsh.Package
	BasePkgName  string
	PkgName      string   // Имя конкретного подпакета (может отличаться от BasePkgName)
	Dependencies []string // Имена runtime зависимостей (depends)
	BuildDeps    []string // Имена зависимостей для сборки (build_deps)
	IsTarget     bool     // true если это целевой пакет (запрошен пользователем)
}

// ResolveDependencyTree рекурсивно разрешает все зависимости и возвращает
// плоский список всех уникальных пакетов, необходимых для сборки
// и список системных зависимостей (не найденных в ALR-репозиториях)
func (b *Builder) ResolveDependencyTree(
	ctx context.Context,
	input interface {
		OsInfoProvider
		PkgFormatProvider
	},
	initialPkgs []string,
) (map[string]*DependencyNode, []string, error) {
	resolved := make(map[string]*DependencyNode)
	visited := make(map[string]bool)
	systemDeps := make(map[string]bool)

	overrideNames, err := overrides.Resolve(input.OSRelease(), overrides.DefaultOpts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve overrides: %w", err)
	}

	var resolve func(pkgNames []string) error
	resolve = func(pkgNames []string) error {
		if len(pkgNames) == 0 {
			return nil
		}

		// Находим пакеты
		found, notFound, err := b.repos.FindPkgs(ctx, pkgNames)
		if err != nil {
			return fmt.Errorf("failed to find packages: %w", err)
		}

		// При preferALRDeps=false: для пакетов, найденных в ALR, проверяем
		// наличие в системном репо. Если пакет доступен в системном репо —
		// предпочитаем системный пакет.
		if b.cfg != nil && !b.cfg.PreferALRDeps() && b.mgr != nil {
			for pkgName := range found {
				available, err := b.mgr.ListAvailable(pkgName)
				if err != nil {
					continue
				}
				for _, av := range available {
					if av == pkgName {
						slog.Debug("Preferring system package over ALR", "pkg", pkgName)
						delete(found, pkgName)
						notFound = append(notFound, pkgName)
						break
					}
				}
			}
		}

		// Собираем системные зависимости (не найденные в ALR)
		for _, pkgName := range notFound {
			systemDeps[pkgName] = true
		}

		// Обрабатываем найденные пакеты
		for pkgName, pkgList := range found {
			if visited[pkgName] {
				continue
			}
			visited[pkgName] = true

			// Берем первый пакет из списка (или можно добавить выбор пользователя)
			if len(pkgList) == 0 {
				continue
			}

			pkg := pkgList[0]

			alrsh.ResolvePackage(&pkg, overrideNames)

			baseName := pkg.BasePkgName
			if baseName == "" {
				baseName = pkg.Name
			}

			if resolved[pkgName] != nil {
				continue
			}

			deps := pkg.Depends.Resolved()
			buildDeps := pkg.BuildDepends.Resolved()

			// Добавляем узел в resolved с ключом = имя подпакета
			resolved[pkgName] = &DependencyNode{
				Package:      &pkg,
				BasePkgName:  baseName,
				PkgName:      pkgName,
				Dependencies: deps,
				BuildDeps:    buildDeps,
			}

			// Объединяем все зависимости для рекурсивного разрешения
			allDeps := append([]string{}, deps...)
			allDeps = append(allDeps, buildDeps...)

			// Рекурсивно разрешаем зависимости
			if len(allDeps) > 0 {
				if err := resolve(allDeps); err != nil {
					return err
				}
			}
		}

		return nil
	}

	// Начинаем разрешение с начальных пакетов
	if err := resolve(initialPkgs); err != nil {
		return nil, nil, err
	}

	// Преобразуем map в слайс для системных зависимостей
	var systemDepsList []string
	for dep := range systemDeps {
		systemDepsList = append(systemDepsList, dep)
	}

	return resolved, systemDepsList, nil
}

// TopologicalSort выполняет топологическую сортировку пакетов по зависимостям
// Возвращает список имен подпакетов в порядке сборки (от корней к листьям)
// Учитывает как runtime depends, так и build_deps
func TopologicalSort(nodes map[string]*DependencyNode) ([]string, error) {
	// Список для результата
	var result []string

	// Множество посещенных узлов
	visited := make(map[string]bool)

	// Множество узлов в текущем пути (для обнаружения циклов)
	inStack := make(map[string]bool)

	var visit func(pkgName string) error
	visit = func(pkgName string) error {
		if visited[pkgName] {
			return nil
		}

		if inStack[pkgName] {
			return fmt.Errorf("circular dependency detected: %s", pkgName)
		}

		node := nodes[pkgName]
		if node == nil {
			// Это системный пакет или пакет не в дереве, игнорируем
			return nil
		}

		inStack[pkgName] = true

		// Посещаем все зависимости (runtime + build)
		allDeps := append([]string{}, node.Dependencies...)
		allDeps = append(allDeps, node.BuildDeps...)

		for _, dep := range allDeps {
			// Используем имя зависимости напрямую (это имя подпакета)
			if err := visit(dep); err != nil {
				return err
			}
		}

		inStack[pkgName] = false
		visited[pkgName] = true
		result = append(result, pkgName)

		return nil
	}

	// Посещаем все узлы
	for pkgName := range nodes {
		if err := visit(pkgName); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// MarkTargetPackages помечает указанные пакеты как целевые (запрошенные пользователем)
func MarkTargetPackages(nodes map[string]*DependencyNode, targetPkgNames []string) {
	for _, name := range targetPkgNames {
		if node, ok := nodes[name]; ok {
			node.IsTarget = true
		}
	}
}

// GetAllDependencies возвращает полный список всех зависимостей (runtime + build)
// для всех пакетов в дереве, исключая целевые пакеты
func GetAllDependencies(nodes map[string]*DependencyNode) []string {
	depMap := make(map[string]bool)

	for _, node := range nodes {
		if node.IsTarget {
			// Для целевых пакетов не включаем их зависимости в общий список
			// они будут обработаны отдельно
			continue
		}

		// Добавляем все зависимости (runtime + build)
		for _, dep := range node.Dependencies {
			depMap[dep] = true
		}
		for _, dep := range node.BuildDeps {
			depMap[dep] = true
		}
	}

	var result []string
	for dep := range depMap {
		result = append(result, dep)
	}

	return result
}

// GetTargetPackages возвращает список целевых пакетов
func GetTargetPackages(nodes map[string]*DependencyNode) []*DependencyNode {
	var result []*DependencyNode
	for _, node := range nodes {
		if node.IsTarget {
			result = append(result, node)
		}
	}
	return result
}

// GetDependencyOnlyPackages возвращает список пакетов-зависимостей (не целевых)
func GetDependencyOnlyPackages(nodes map[string]*DependencyNode) []*DependencyNode {
	var result []*DependencyNode
	for _, node := range nodes {
		if !node.IsTarget {
			result = append(result, node)
		}
	}
	return result
}
