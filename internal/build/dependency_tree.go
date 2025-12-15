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

	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/alrsh"
)

// DependencyNode представляет узел в дереве зависимостей
type DependencyNode struct {
	Package      *alrsh.Package
	BasePkgName  string
	PkgName      string   // Имя конкретного подпакета (может отличаться от BasePkgName)
	Dependencies []string // Имена зависимостей
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
	systemDeps := make(map[string]bool) // Для дедупликации системных зависимостей

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

			// Определяем базовое имя пакета
			baseName := pkg.BasePkgName
			if baseName == "" {
				baseName = pkg.Name
			}

			// Используем имя конкретного подпакета как ключ (не basePkgName)
			// Это позволяет собирать только запрошенный подпакет, а не весь мультипакет
			if resolved[pkgName] != nil {
				continue
			}

			// Получаем зависимости для этого дистрибутива
			// Пакет из БД уже содержит разрешенные значения для текущего дистрибутива
			deps := pkg.Depends.Resolved()
			buildDeps := pkg.BuildDepends.Resolved()

			// Объединяем зависимости
			allDeps := append([]string{}, deps...)
			allDeps = append(allDeps, buildDeps...)

			// Добавляем узел в resolved с ключом = имя подпакета
			resolved[pkgName] = &DependencyNode{
				Package:      &pkg,
				BasePkgName:  baseName,
				PkgName:      pkgName,
				Dependencies: allDeps,
			}

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

		// Посещаем все зависимости
		for _, dep := range node.Dependencies {
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
