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

package build

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"log/slog"

	"github.com/leonelquinteros/gotext"

	"git.alr-pkg.ru/Plemya-x/ALR/internal/cliutils"
	"git.alr-pkg.ru/Plemya-x/ALR/internal/config"
	"git.alr-pkg.ru/Plemya-x/ALR/internal/manager"
	"git.alr-pkg.ru/Plemya-x/ALR/internal/stats"
	"git.alr-pkg.ru/Plemya-x/ALR/pkg/alrsh"
	"git.alr-pkg.ru/Plemya-x/ALR/pkg/distro"
	"git.alr-pkg.ru/Plemya-x/ALR/pkg/types"
)

type BuildInput struct {
	opts             *types.BuildOpts
	info             *distro.OSRelease
	pkgFormat        string
	script           string
	repository       string
	packages         []string
	skipDepsBuilding bool // Пропустить сборку зависимостей (используется при вызове из BuildALRDeps)
}

func (bi *BuildInput) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)

	if err := encoder.Encode(bi.opts); err != nil {
		return nil, err
	}
	if err := encoder.Encode(bi.info); err != nil {
		return nil, err
	}
	if err := encoder.Encode(bi.pkgFormat); err != nil {
		return nil, err
	}
	if err := encoder.Encode(bi.script); err != nil {
		return nil, err
	}
	if err := encoder.Encode(bi.repository); err != nil {
		return nil, err
	}
	if err := encoder.Encode(bi.packages); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

func (bi *BuildInput) GobDecode(data []byte) error {
	r := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(r)

	if err := decoder.Decode(&bi.opts); err != nil {
		return err
	}
	if err := decoder.Decode(&bi.info); err != nil {
		return err
	}
	if err := decoder.Decode(&bi.pkgFormat); err != nil {
		return err
	}
	if err := decoder.Decode(&bi.script); err != nil {
		return err
	}
	if err := decoder.Decode(&bi.repository); err != nil {
		return err
	}
	if err := decoder.Decode(&bi.packages); err != nil {
		return err
	}

	return nil
}

func (b *BuildInput) Repository() string {
	return b.repository
}

func (b *BuildInput) BuildOpts() *types.BuildOpts {
	return b.opts
}

func (b *BuildInput) OSRelease() *distro.OSRelease {
	return b.info
}

func (b *BuildInput) PkgFormat() string {
	return b.pkgFormat
}

type BuildOptsProvider interface {
	BuildOpts() *types.BuildOpts
}

type OsInfoProvider interface {
	OSRelease() *distro.OSRelease
}

type PkgFormatProvider interface {
	PkgFormat() string
}

type RepositoryProvider interface {
	Repository() string
}

// ================================================

type BuiltDep struct {
	Name string
	Path string
}

func Map[T, R any](items []T, f func(T) R) []R {
	res := make([]R, len(items))
	for i, item := range items {
		res[i] = f(item)
	}
	return res
}

func GetBuiltPaths(deps []*BuiltDep) []string {
	return Map(deps, func(dep *BuiltDep) string {
		return dep.Path
	})
}

func GetBuiltName(deps []*BuiltDep) []string {
	return Map(deps, func(dep *BuiltDep) string {
		return dep.Name
	})
}

type PackageFinder interface {
	FindPkgs(ctx context.Context, pkgs []string) (map[string][]alrsh.Package, []string, error)
}

type Config interface {
	GetPaths() *config.Paths
	PagerStyle() string
	PreferALRDeps() bool
}

type FunctionsOutput struct {
	Contents *[]string
}

// EXECUTORS

type ScriptResolverExecutor interface {
	ResolveScript(ctx context.Context, pkg *alrsh.Package) *ScriptInfo
}

type CacheExecutor interface {
	CheckForBuiltPackage(ctx context.Context, input *BuildInput, vars *alrsh.Package) (string, bool, error)
}

type ScriptViewerExecutor interface {
	ViewScript(ctx context.Context, input *BuildInput, sf *alrsh.ScriptFile, basePkg string) error
}

type CheckerExecutor interface {
	PerformChecks(
		ctx context.Context,
		input *BuildInput,
		vars *alrsh.Package,
	) (bool, error)
}

type SourcesInput struct {
	Sources   []string
	Checksums []string
}

type SourceDownloaderExecutor interface {
	DownloadSources(
		ctx context.Context,
		input *BuildInput,
		basePkg string,
		si SourcesInput,
	) error
}

//

func NewBuilder(
	scriptResolver ScriptResolverExecutor,
	scriptExecutor ScriptExecutor,
	cacheExecutor CacheExecutor,
	scriptViewerExecutor ScriptViewerExecutor,
	checkerExecutor CheckerExecutor,
	installerExecutor InstallerExecutor,
	sourceExecutor SourceDownloaderExecutor,
) *Builder {
	return &Builder{
		scriptResolver:       scriptResolver,
		scriptExecutor:       scriptExecutor,
		cacheExecutor:        cacheExecutor,
		scriptViewerExecutor: scriptViewerExecutor,
		checkerExecutor:      checkerExecutor,
		installerExecutor:    installerExecutor,
		sourceExecutor:       sourceExecutor,
	}
}

type Builder struct {
	scriptResolver       ScriptResolverExecutor
	scriptExecutor       ScriptExecutor
	cacheExecutor        CacheExecutor
	scriptViewerExecutor ScriptViewerExecutor
	checkerExecutor      CheckerExecutor
	installerExecutor    InstallerExecutor
	sourceExecutor       SourceDownloaderExecutor
	repos                PackageFinder
	mgr                  manager.Manager
	cfg                  Config
}

type BuildArgs struct {
	Opts             *types.BuildOpts
	Info             *distro.OSRelease
	PkgFormat_       string
	SkipDepsBuilding bool // Пропустить сборку зависимостей (используется при вызове из BuildALRDeps)
}

func (b *BuildArgs) BuildOpts() *types.BuildOpts {
	return b.Opts
}

func (b *BuildArgs) OSRelease() *distro.OSRelease {
	return b.Info
}

func (b *BuildArgs) PkgFormat() string {
	return b.PkgFormat_
}

type BuildPackageFromDbArgs struct {
	BuildArgs
	Package  *alrsh.Package
	Packages []string
}

type BuildPackageFromScriptArgs struct {
	BuildArgs
	Script   string
	Packages []string
}

func (b *Builder) BuildPackageFromDb(
	ctx context.Context,
	args *BuildPackageFromDbArgs,
) ([]*BuiltDep, error) {
	scriptInfo := b.scriptResolver.ResolveScript(ctx, args.Package)

	return b.BuildPackage(ctx, &BuildInput{
		script:           scriptInfo.Script,
		repository:       scriptInfo.Repository,
		packages:         args.Packages,
		pkgFormat:        args.PkgFormat(),
		opts:             args.Opts,
		info:             args.Info,
		skipDepsBuilding: args.SkipDepsBuilding,
	})
}

func (b *Builder) BuildPackageFromScript(
	ctx context.Context,
	args *BuildPackageFromScriptArgs,
) ([]*BuiltDep, error) {
	return b.BuildPackage(ctx, &BuildInput{
		script:     args.Script,
		repository: ExtractRepoNameFromPath(args.Script),
		packages:   args.Packages,
		pkgFormat:  args.PkgFormat(),
		opts:       args.Opts,
		info:       args.Info,
	})
}

func (b *Builder) BuildPackage(
	ctx context.Context,
	input *BuildInput,
) ([]*BuiltDep, error) {
	scriptPath := input.script

	slog.Debug("ReadScript")
	sf, err := b.scriptExecutor.ReadScript(ctx, scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed reading script: %w", err)
	}

	slog.Debug("ExecuteFirstPass")
	basePkg, varsOfPackages, err := b.scriptExecutor.ExecuteFirstPass(ctx, input, sf)
	if err != nil {
		return nil, fmt.Errorf("failed ExecuteFirstPass: %w", err)
	}

	var builtDeps []*BuiltDep
	var remainingVars []*alrsh.Package

	if !input.opts.Clean {
		for _, vars := range varsOfPackages {
			builtPkgPath, ok, err := b.cacheExecutor.CheckForBuiltPackage(ctx, input, vars)
			if err != nil {
				return nil, err
			}
			if ok {
				builtDeps = append(builtDeps, &BuiltDep{
					Path: builtPkgPath,
					Name: vars.Name,
				})
			} else {
				remainingVars = append(remainingVars, vars)
			}
		}

		if len(remainingVars) == 0 {
			slog.Info(gotext.Get("Using cached package"), "name", basePkg)
			return builtDeps, nil
		}

		// Обновляем varsOfPackages только теми пакетами, которые нужно собрать
		varsOfPackages = remainingVars
	}

	slog.Debug("ViewScript")
	slog.Debug("", "varsOfPackages", varsOfPackages[0])
	err = b.scriptViewerExecutor.ViewScript(ctx, input, sf, basePkg)
	if err != nil {
		return nil, err
	}

	slog.Info(gotext.Get("Building package"), "name", basePkg)

	for _, vars := range varsOfPackages {
		cont, err := b.checkerExecutor.PerformChecks(ctx, input, vars)
		if err != nil {
			return nil, err
		}
		if !cont {
			return nil, errors.New("exit...")
		}
	}

	buildDepends := []string{}
	optDepends := []string{}
	depends := []string{}
	sources := []string{}
	checksums := []string{}
	for _, vars := range varsOfPackages {
		buildDepends = append(buildDepends, vars.BuildDepends.Resolved()...)
		optDepends = append(optDepends, vars.OptDepends.Resolved()...)
		depends = append(depends, vars.Depends.Resolved()...)
		sources = append(sources, vars.Sources.Resolved()...)
		checksums = append(checksums, vars.Checksums.Resolved()...)
	}
	buildDepends = removeDuplicates(buildDepends)
	optDepends = removeDuplicates(optDepends)
	depends = removeDuplicates(depends)

	if len(sources) != len(checksums) {
		slog.Error(gotext.Get("The checksums array must be the same length as sources"))
		return nil, errors.New("exit...")
	}
	sources, checksums = removeDuplicatesSources(sources, checksums)

	slog.Debug("installBuildDeps")
	// build_deps всегда устанавливаются - они нужны для сборки
	alrBuildDeps, installedBuildDeps, err := b.installBuildDeps(ctx, input, buildDepends)
	if err != nil {
		return nil, err
	}

	slog.Debug("installOptDeps")
	_, err = b.installOptDeps(ctx, input, optDepends)
	if err != nil {
		return nil, err
	}

	depNames := make(map[string]struct{})
	for _, dep := range alrBuildDeps {
		depNames[dep.Name] = struct{}{}
	}

	// We filter so as not to re-build what has already been built at the `installBuildDeps` stage.
	var filteredDepends []string

	// Создаем набор подпакетов текущего мультипакета для исключения циклических зависимостей
	// Используем имена из varsOfPackages, так как input.packages может быть пустым
	currentPackageNames := make(map[string]struct{})
	for _, vars := range varsOfPackages {
		currentPackageNames[vars.Name] = struct{}{}
	}

	for _, d := range depends {
		if _, found := depNames[d]; !found {
			// Исключаем зависимости, которые являются подпакетами текущего мультипакета
			if _, isCurrentPackage := currentPackageNames[d]; !isCurrentPackage {
				filteredDepends = append(filteredDepends, d)
			}
		}
	}

	var newBuiltDeps []*BuiltDep
	var repoDeps []string

	// Пропускаем сборку зависимостей если флаг установлен (вызов из BuildALRDeps)
	if !input.skipDepsBuilding {
		slog.Debug("BuildALRDeps")
		newBuiltDeps, repoDeps, err = b.BuildALRDeps(ctx, input, filteredDepends)
		if err != nil {
			return nil, err
		}
	}

	slog.Debug("PrepareDirs")
	err = b.scriptExecutor.PrepareDirs(ctx, input, basePkg)
	if err != nil {
		return nil, err
	}

	slog.Info(gotext.Get("Downloading sources"))
	slog.Debug("DownloadSources")
	err = b.sourceExecutor.DownloadSources(
		ctx,
		input,
		basePkg,
		SourcesInput{
			Sources:   sources,
			Checksums: checksums,
		},
	)
	if err != nil {
		return nil, err
	}

	builtDeps = removeDuplicates(append(builtDeps, newBuiltDeps...))

	slog.Debug("ExecuteSecondPass")
	res, err := b.scriptExecutor.ExecuteSecondPass(
		ctx,
		input,
		sf,
		varsOfPackages,
		repoDeps,
		builtDeps,
		basePkg,
	)
	if err != nil {
		return nil, err
	}

	builtDeps = removeDuplicates(append(builtDeps, res...))

	// Удаляем build_deps после сборки
	if len(installedBuildDeps) > 0 {
		err = b.removeBuildDeps(ctx, input, installedBuildDeps)
		if err != nil {
			return nil, err
		}
	}

	return builtDeps, nil
}

func (b *Builder) removeBuildDeps(ctx context.Context, input interface {
	BuildOptsProvider
}, deps []string,
) error {
	if len(deps) > 0 {
		remove, err := cliutils.YesNoPrompt(ctx, gotext.Get("Would you like to remove the build dependencies?"), input.BuildOpts().Interactive, false)
		if err != nil {
			return err
		}

		if remove {
			err = b.installerExecutor.Remove(
				ctx,
				deps,
				&manager.Opts{
					NoConfirm: !input.BuildOpts().Interactive,
				},
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type InstallPkgsArgs struct {
	BuildArgs
	AlrPkgs    []alrsh.Package
	NativePkgs []string
}

func (b *Builder) InstallALRPackages(
	ctx context.Context,
	input interface {
		OsInfoProvider
		BuildOptsProvider
		PkgFormatProvider
	},
	alrPkgs []alrsh.Package,
) error {
	if len(alrPkgs) == 0 {
		return nil
	}

	// Получаем имена всех пакетов для установки
	pkgNames := make([]string, len(alrPkgs))
	for i, pkg := range alrPkgs {
		pkgNames[i] = pkg.Name
	}

	// Используем тот же алгоритм что и InstallPkgs
	_, err := b.InstallPkgs(ctx, input, pkgNames)
	return err
}

func (b *Builder) BuildALRDeps(
	ctx context.Context,
	input interface {
		OsInfoProvider
		BuildOptsProvider
		PkgFormatProvider
	},
	depends []string,
) (buildDeps []*BuiltDep, repoDeps []string, err error) {
	if len(depends) == 0 {
		return nil, nil, nil
	}

	slog.Info(gotext.Get("Installing dependencies"))

	// Используем новое дерево зависимостей
	tree, err := b.ResolveDependencyTreeV2(ctx, input, depends)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve dependency tree: %w", err)
	}

	// Системные зависимости
	allSystemDeps := append([]string{}, tree.SystemBuildDeps...)
	allSystemDeps = append(allSystemDeps, tree.SystemRuntimeDeps...)
	repoDeps = removeDuplicates(allSystemDeps)

	var allBuiltDeps []*BuiltDep

	// ФАЗА 1: Собираем build зависимости
	if len(tree.BuildDepsOrder) > 0 {
		slog.Info(gotext.Get("Installing build dependencies"))

		// Сначала системные build зависимости
		if len(tree.SystemBuildDeps) > 0 {
			err = b.installerExecutor.Install(ctx, tree.SystemBuildDeps, &manager.Opts{
				NoConfirm: !input.BuildOpts().Interactive,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("failed to install system build dependencies: %w", err)
			}
		}

		// Затем ALR build зависимости
		built, err := b.BuildAndInstallFromOrder(ctx, input, tree, tree.BuildDepsOrder)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build dependencies: %w", err)
		}
		allBuiltDeps = append(allBuiltDeps, built...)
	}

	// ФАЗА 2: Собираем runtime зависимости
	if len(tree.RuntimeDepsOrder) > 0 {
		slog.Info(gotext.Get("Installing runtime dependencies"))

		// Сначала системные runtime зависимости
		if len(tree.SystemRuntimeDeps) > 0 {
			err = b.installerExecutor.Install(ctx, tree.SystemRuntimeDeps, &manager.Opts{
				NoConfirm: !input.BuildOpts().Interactive,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("failed to install system runtime dependencies: %w", err)
			}
		}

		// Затем ALR runtime зависимости
		built, err := b.BuildAndInstallFromOrder(ctx, input, tree, tree.RuntimeDepsOrder)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build dependencies: %w", err)
		}
		allBuiltDeps = append(allBuiltDeps, built...)
	}

	return allBuiltDeps, repoDeps, nil
}

// BuildALRDepsFromTree собирает зависимости из уже построенного дерева
// Используется для фазы 1 установки - установка всех зависимостей перед сборкой целевых пакетов
func (b *Builder) BuildALRDepsFromTree(
	ctx context.Context,
	input interface {
		OsInfoProvider
		BuildOptsProvider
		PkgFormatProvider
	},
	depTree map[string]*DependencyNode,
	depPkgNames []string,
) (buildDeps []*BuiltDep, repoDeps []string, err error) {
	if len(depPkgNames) == 0 {
		return nil, nil, nil
	}

	slog.Debug("Building dependencies from tree", "count", len(depPkgNames))

	// Топологическая сортировка зависимостей (от листьев к корню)
	sortedPkgs, err := TopologicalSort(depTree)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sort dependencies: %w", err)
	}

	// Фильтруем только пакеты-зависимости (не целевые)
	var sortedDeps []string
	for _, pkgName := range sortedPkgs {
		if node, ok := depTree[pkgName]; ok && !node.IsTarget {
			sortedDeps = append(sortedDeps, pkgName)
		}
	}

	slog.Debug("Sorted dependencies", "order", sortedDeps)

	// Получаем список пакетов, которые нужно собрать (не установлены или устарели)
	var allPkgs []alrsh.Package
	for _, node := range depTree {
		if !node.IsTarget && node.Package != nil {
			allPkgs = append(allPkgs, *node.Package)
		}
	}

	needBuildPkgs, err := b.installerExecutor.FilterPackagesByVersion(ctx, allPkgs, input.OSRelease())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to filter packages: %w", err)
	}

	needBuildNames := make(map[string]bool)
	for _, pkg := range needBuildPkgs {
		needBuildNames[pkg.Name] = true
	}

	// Собираем зависимости в правильном порядке
	for _, pkgName := range sortedDeps {
		node := depTree[pkgName]
		if node == nil || node.Package == nil {
			continue
		}

		pkg := node.Package
		basePkgName := node.BasePkgName

		// Пропускаем уже установленные пакеты
		if !needBuildNames[pkg.Name] {
			slog.Debug("Package already installed, skipping", "name", pkg.Name)
			continue
		}

		// Проверяем кеш
		scriptInfo := b.scriptResolver.ResolveScript(ctx, pkg)
		buildInput := &BuildInput{
			script:     scriptInfo.Script,
			repository: scriptInfo.Repository,
			packages:   []string{pkgName},
			pkgFormat:  input.PkgFormat(),
			opts:       input.BuildOpts(),
			info:       input.OSRelease(),
		}

		cachedDeps, allInCache, err := b.checkCacheForAllSubpackages(ctx, buildInput, basePkgName, []string{pkgName})
		if err != nil {
			return nil, nil, err
		}

		if allInCache {
			slog.Debug("Using cached package", "name", pkgName)
			buildDeps = append(buildDeps, cachedDeps...)
			continue
		}

		// Устанавливаем системные зависимости для этого пакета (build_deps)
		if len(node.SystemDeps) > 0 {
			slog.Info(gotext.Get("Installing system build dependencies"), "package", pkgName, "deps", node.SystemDeps)
			err = b.installerExecutor.Install(ctx, node.SystemDeps, &manager.Opts{
				NoConfirm: !input.BuildOpts().Interactive,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("failed to install system dependencies for %s: %w", pkgName, err)
			}
		}

		// Собираем пакет
		slog.Info(gotext.Get("Building dependency"), "name", pkgName)

		res, err := b.BuildPackageFromDb(
			ctx,
			&BuildPackageFromDbArgs{
				Package:  pkg,
				Packages: []string{pkgName},
				BuildArgs: BuildArgs{
					Opts:             input.BuildOpts(),
					Info:             input.OSRelease(),
					PkgFormat_:       input.PkgFormat(),
					SkipDepsBuilding: true, // ALR зависимости уже собраны
				},
			},
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build dependency %s: %w", pkgName, err)
		}

		buildDeps = append(buildDeps, res...)

		// Устанавливаем собранную зависимость сразу, чтобы она была доступна для следующих
		if len(res) > 0 {
			err = b.installerExecutor.InstallLocal(ctx, GetBuiltPaths(res), &manager.Opts{
				NoConfirm: !input.BuildOpts().Interactive,
			})
			if err != nil {
				return nil, nil, fmt.Errorf("failed to install dependency %s: %w", pkgName, err)
			}
		}
	}

	buildDeps = removeDuplicates(buildDeps)

	return buildDeps, nil, nil
}

// BuildTargetPackages собирает целевые пакеты после установки всех зависимостей
func (b *Builder) BuildTargetPackages(
	ctx context.Context,
	input interface {
		OsInfoProvider
		BuildOptsProvider
		PkgFormatProvider
	},
	depTree map[string]*DependencyNode,
	targetNodes []*DependencyNode,
) ([]*BuiltDep, error) {
	if len(targetNodes) == 0 {
		return nil, nil
	}

	// Топологическая сортировка всех пакетов для правильного порядка сборки
	sortedPkgs, err := TopologicalSort(depTree)
	if err != nil {
		return nil, fmt.Errorf("failed to sort packages: %w", err)
	}

	// Фильтруем только целевые пакеты в порядке сортировки
	targetNames := make(map[string]bool)
	for _, node := range targetNodes {
		targetNames[node.PkgName] = true
	}

	var sortedTargets []string
	for _, pkgName := range sortedPkgs {
		if targetNames[pkgName] {
			sortedTargets = append(sortedTargets, pkgName)
		}
	}

	var allBuiltDeps []*BuiltDep

	// Собираем целевые пакеты в правильном порядке
	for _, pkgName := range sortedTargets {
		node := depTree[pkgName]
		if node == nil || node.Package == nil {
			continue
		}

		pkg := node.Package
		basePkgName := node.BasePkgName

		// Проверяем нужна ли сборка
		needBuildPkgs, err := b.installerExecutor.FilterPackagesByVersion(ctx, []alrsh.Package{*pkg}, input.OSRelease())
		if err != nil {
			return nil, fmt.Errorf("failed to filter package %s: %w", pkgName, err)
		}

		if len(needBuildPkgs) == 0 {
			slog.Info(gotext.Get("Package is already installed"), "name", pkgName)
			continue
		}

		// Проверяем кеш
		scriptInfo := b.scriptResolver.ResolveScript(ctx, pkg)
		buildInput := &BuildInput{
			script:     scriptInfo.Script,
			repository: scriptInfo.Repository,
			packages:   []string{pkgName},
			pkgFormat:  input.PkgFormat(),
			opts:       input.BuildOpts(),
			info:       input.OSRelease(),
		}

		cachedDeps, allInCache, err := b.checkCacheForAllSubpackages(ctx, buildInput, basePkgName, []string{pkgName})
		if err != nil {
			return nil, err
		}

		if allInCache {
			slog.Info(gotext.Get("Using cached package"), "name", pkgName)
			allBuiltDeps = append(allBuiltDeps, cachedDeps...)
			continue
		}

		// Устанавливаем системные зависимости для целевого пакета
		if len(node.SystemDeps) > 0 {
			slog.Info(gotext.Get("Installing system build dependencies"), "package", pkgName, "deps", node.SystemDeps)
			err = b.installerExecutor.Install(ctx, node.SystemDeps, &manager.Opts{
				NoConfirm: !input.BuildOpts().Interactive,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to install system dependencies for %s: %w", pkgName, err)
			}
		}

		// Собираем целевой пакет
		slog.Info(gotext.Get("Building package"), "name", pkgName)

		res, err := b.BuildPackageFromDb(
			ctx,
			&BuildPackageFromDbArgs{
				Package:  pkg,
				Packages: []string{pkgName},
				BuildArgs: BuildArgs{
					Opts:             input.BuildOpts(),
					Info:             input.OSRelease(),
					PkgFormat_:       input.PkgFormat(),
					SkipDepsBuilding: true, // Все зависимости уже установлены
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build package %s: %w", pkgName, err)
		}

		allBuiltDeps = append(allBuiltDeps, res...)
	}

	return removeDuplicates(allBuiltDeps), nil
}

// findAllSubpackages находит все подпакеты для базового пакета
func (b *Builder) findAllSubpackages(ctx context.Context, basePkgName, repository string) ([]string, error) {
	// Запрашиваем все пакеты с этим basepkg_name
	pkgs, _, err := b.repos.FindPkgs(ctx, []string{basePkgName})
	if err != nil {
		return nil, err
	}

	var subpkgs []string
	seen := make(map[string]bool)

	for _, pkgList := range pkgs {
		for _, pkg := range pkgList {
			// Проверяем, что это пакет из нужного репозитория
			if pkg.Repository == repository {
				pkgBase := pkg.BasePkgName
				if pkgBase == "" {
					pkgBase = pkg.Name
				}

				// Добавляем только если это пакет с нужным BasePkgName
				if pkgBase == basePkgName && !seen[pkg.Name] {
					subpkgs = append(subpkgs, pkg.Name)
					seen[pkg.Name] = true
				}
			}
		}
	}

	return subpkgs, nil
}

// checkCacheForAllSubpackages проверяет кеш для всех подпакетов
func (b *Builder) checkCacheForAllSubpackages(
	ctx context.Context,
	buildInput *BuildInput,
	basePkgName string,
	subpkgs []string,
) ([]*BuiltDep, bool, error) {
	var cachedDeps []*BuiltDep
	allInCache := true

	// Получаем информацию обо всех подпакетах
	pkgsInfo, _, err := b.repos.FindPkgs(ctx, subpkgs)
	if err != nil {
		return nil, false, fmt.Errorf("failed to find subpackages info: %w", err)
	}

	for _, pkgName := range subpkgs {
		var pkgForCheck *alrsh.Package

		// Находим Package для подпакета
		if pkgList, ok := pkgsInfo[pkgName]; ok && len(pkgList) > 0 {
			pkgForCheck = &pkgList[0]
		}

		if pkgForCheck != nil {
			pkgPath, found, err := b.cacheExecutor.CheckForBuiltPackage(ctx, buildInput, pkgForCheck)
			if err != nil {
				return nil, false, fmt.Errorf("failed to check cache: %w", err)
			}

			if found {
				slog.Info(gotext.Get("Using cached package"), "name", pkgName, "path", pkgPath)
				cachedDeps = append(cachedDeps, &BuiltDep{
					Name: pkgName,
					Path: pkgPath,
				})
			} else {
				allInCache = false
				break
			}
		}
	}

	return cachedDeps, allInCache && len(cachedDeps) > 0, nil
}

func (i *Builder) installBuildDeps(
	ctx context.Context,
	input interface {
		OsInfoProvider
		BuildOptsProvider
		PkgFormatProvider
	},
	pkgs []string,
) ([]*BuiltDep, []string, error) {
	var builtDeps []*BuiltDep
	var deps []string
	var err error
	if len(pkgs) > 0 {
		deps, err = i.installerExecutor.RemoveAlreadyInstalled(ctx, pkgs)
		if err != nil {
			return nil, nil, err
		}

		builtDeps, err = i.InstallPkgs(ctx, input, deps) // Устанавливаем выбранные пакеты
		if err != nil {
			return nil, nil, err
		}
	}
	return builtDeps, deps, nil
}

func (i *Builder) installOptDeps(
	ctx context.Context,
	input interface {
		OsInfoProvider
		BuildOptsProvider
		PkgFormatProvider
	},
	pkgs []string,
) ([]*BuiltDep, error) {
	var builtDeps []*BuiltDep
	optDeps, err := i.installerExecutor.RemoveAlreadyInstalled(ctx, pkgs)
	if err != nil {
		return nil, err
	}
	if len(optDeps) > 0 {
		optDeps, err := cliutils.ChooseOptDepends(
			ctx,
			optDeps,
			"install",
			input.BuildOpts().Interactive,
		) // Пользователя просят выбрать опциональные зависимости
		if err != nil {
			return nil, err
		}

		if len(optDeps) == 0 {
			return builtDeps, nil
		}

		builtDeps, err = i.InstallPkgs(ctx, input, optDeps) // Устанавливаем выбранные пакеты
		if err != nil {
			return nil, err
		}
	}
	return builtDeps, nil
}

func (i *Builder) InstallPkgs(
	ctx context.Context,
	input interface {
		OsInfoProvider
		BuildOptsProvider
		PkgFormatProvider
	},
	pkgs []string,
) ([]*BuiltDep, error) {
	if len(pkgs) == 0 {
		return nil, nil
	}

	slog.Info(gotext.Get("Resolving dependencies for packages"), "packages", pkgs)

	// Шаг 1: Построить полное дерево зависимостей
	tree, err := i.ResolveDependencyTreeV2(ctx, input, pkgs)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependency tree: %w", err)
	}

	slog.Debug("Dependency tree resolved",
		"build_deps", len(tree.BuildDepsOrder),
		"runtime_deps", len(tree.RuntimeDepsOrder),
		"system_build_deps", len(tree.SystemBuildDeps),
		"system_runtime_deps", len(tree.SystemRuntimeDeps),
		"opt_deps", len(tree.AllOptDeps))

	var allBuiltDeps []*BuiltDep

	// ФАЗА 1: Устанавливаем все build зависимости (начиная с самых дальних)
	if len(tree.BuildDepsOrder) > 0 || len(tree.SystemBuildDeps) > 0 {
		slog.Info(gotext.Get("Installing build dependencies"))

		// Сначала системные build зависимости
		if len(tree.SystemBuildDeps) > 0 {
			slog.Info(gotext.Get("Installing system build dependencies"), "count", len(tree.SystemBuildDeps))
			err = i.installerExecutor.Install(ctx, tree.SystemBuildDeps, &manager.Opts{
				NoConfirm: !input.BuildOpts().Interactive,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to install system build dependencies: %w", err)
			}
		}

		// Затем ALR build зависимости (уже в правильном порядке - от листьев к корню)
		if len(tree.BuildDepsOrder) > 0 {
			buildDeps, err := i.BuildAndInstallFromOrder(ctx, input, tree, tree.BuildDepsOrder)
			if err != nil {
				return nil, fmt.Errorf("failed to install build dependencies: %w", err)
			}
			allBuiltDeps = append(allBuiltDeps, buildDeps...)
		}
	}

	// ФАЗА 2: Устанавливаем все runtime зависимости
	if len(tree.RuntimeDepsOrder) > 0 || len(tree.SystemRuntimeDeps) > 0 {
		slog.Info(gotext.Get("Installing runtime dependencies"))

		// Сначала системные runtime зависимости
		if len(tree.SystemRuntimeDeps) > 0 {
			slog.Info(gotext.Get("Installing system runtime dependencies"), "count", len(tree.SystemRuntimeDeps))
			err = i.installerExecutor.Install(ctx, tree.SystemRuntimeDeps, &manager.Opts{
				NoConfirm: !input.BuildOpts().Interactive,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to install system runtime dependencies: %w", err)
			}
			_ = i.installerExecutor.CheckVersionsAfterInstall(ctx, tree.SystemRuntimeDeps)
		}

		// Затем ALR runtime зависимости
		if len(tree.RuntimeDepsOrder) > 0 {
			runtimeDeps, err := i.BuildAndInstallFromOrder(ctx, input, tree, tree.RuntimeDepsOrder)
			if err != nil {
				return nil, fmt.Errorf("failed to install runtime dependencies: %w", err)
			}
			allBuiltDeps = append(allBuiltDeps, runtimeDeps...)
		}
	}

	// ФАЗА 3: Обрабатываем опциональные зависимости
	if len(tree.AllOptDeps) > 0 {
		slog.Info(gotext.Get("Processing optional dependencies"))
		_, err := i.installOptDeps(ctx, input, tree.AllOptDeps)
		if err != nil {
			return nil, fmt.Errorf("failed to install optional dependencies: %w", err)
		}
	}

	// ФАЗА 4: Собираем и устанавливаем целевые пакеты
	slog.Info(gotext.Get("Building target packages"), "count", len(pkgs))

	// Помечаем целевые пакеты
	for _, pkgName := range pkgs {
		if node, ok := tree.Nodes[pkgName]; ok {
			node.IsTarget = true
		}
	}

	targetDeps, err := i.BuildAndInstallFromOrder(ctx, input, tree, pkgs)
	if err != nil {
		return nil, fmt.Errorf("failed to build target packages: %w", err)
	}
	allBuiltDeps = append(allBuiltDeps, targetDeps...)

	// Устанавливаем все собранные целевые пакеты
	if len(targetDeps) > 0 {
		slog.Info(gotext.Get("Installing built packages"))
		err = i.installerExecutor.InstallLocal(ctx, GetBuiltPaths(targetDeps), &manager.Opts{
			NoConfirm: !input.BuildOpts().Interactive,
		})
		if err != nil {
			return nil, err
		}

		// Отслеживание установки
		for _, dep := range targetDeps {
			if stats.ShouldTrackPackage(dep.Name) {
				stats.TrackInstallation(ctx, dep.Name, "install")
			}
		}
	}

	return allBuiltDeps, nil
}

// BuildAndInstallFromOrder собирает и устанавливает пакеты в заданном порядке
func (b *Builder) BuildAndInstallFromOrder(
	ctx context.Context,
	input interface {
		OsInfoProvider
		BuildOptsProvider
		PkgFormatProvider
	},
	tree *DependencyTree,
	pkgOrder []string,
) ([]*BuiltDep, error) {
	var allBuiltDeps []*BuiltDep

	for _, pkgName := range pkgOrder {
		node, ok := tree.Nodes[pkgName]
		if !ok || node == nil || node.Package == nil {
			slog.Debug("Package not found in tree, skipping", "name", pkgName)
			continue
		}

		pkg := node.Package
		basePkgName := node.BasePkgName

		// Проверяем нужна ли сборка
		needBuildPkgs, err := b.installerExecutor.FilterPackagesByVersion(ctx, []alrsh.Package{*pkg}, input.OSRelease())
		if err != nil {
			return nil, fmt.Errorf("failed to filter package %s: %w", pkgName, err)
		}

		if len(needBuildPkgs) == 0 && !node.IsTarget {
			slog.Debug("Package already installed, skipping", "name", pkgName)
			continue
		}

		// Проверяем кеш
		scriptInfo := b.scriptResolver.ResolveScript(ctx, pkg)
		buildInput := &BuildInput{
			script:     scriptInfo.Script,
			repository: scriptInfo.Repository,
			packages:   []string{pkgName},
			pkgFormat:  input.PkgFormat(),
			opts:       input.BuildOpts(),
			info:       input.OSRelease(),
		}

		cachedDeps, allInCache, err := b.checkCacheForAllSubpackages(ctx, buildInput, basePkgName, []string{pkgName})
		if err != nil {
			return nil, err
		}

		if allInCache {
			slog.Info(gotext.Get("Using cached package"), "name", pkgName)
			allBuiltDeps = append(allBuiltDeps, cachedDeps...)
			continue
		}

		// Собираем пакет
		if node.IsTarget {
			slog.Info(gotext.Get("Building package"), "name", pkgName)
		} else {
			slog.Info(gotext.Get("Building dependency"), "name", pkgName)
		}

		res, err := b.BuildPackageFromDb(
			ctx,
			&BuildPackageFromDbArgs{
				Package:  pkg,
				Packages: []string{pkgName},
				BuildArgs: BuildArgs{
					Opts:             input.BuildOpts(),
					Info:             input.OSRelease(),
					PkgFormat_:       input.PkgFormat(),
					SkipDepsBuilding: true, // Все зависимости уже установлены
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build %s: %w", pkgName, err)
		}

		allBuiltDeps = append(allBuiltDeps, res...)

		// Устанавливаем собранный пакет сразу, чтобы он был доступен для следующих
		if len(res) > 0 && !node.IsTarget {
			err = b.installerExecutor.InstallLocal(ctx, GetBuiltPaths(res), &manager.Opts{
				NoConfirm: !input.BuildOpts().Interactive,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to install %s: %w", pkgName, err)
			}
		}
	}

	return removeDuplicates(allBuiltDeps), nil
}
