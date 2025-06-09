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
	"log/slog"

	"github.com/leonelquinteros/gotext"
	"mvdan.cc/sh/v3/syntax"
	"mvdan.cc/sh/v3/syntax/typedjson"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/distro"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/manager"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/types"
)

type BuildInput struct {
	opts       *types.BuildOpts
	info       *distro.OSRelease
	pkgFormat  string
	script     string
	repository string
	packages   []string
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

type ScriptFile struct {
	File *syntax.File
	Path string
}

func (s *ScriptFile) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(s.Path); err != nil {
		return nil, err
	}
	var fileBuf bytes.Buffer
	if err := typedjson.Encode(&fileBuf, s.File); err != nil {
		return nil, err
	}
	fileData := fileBuf.Bytes()
	if err := enc.Encode(fileData); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ScriptFile) GobDecode(data []byte) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&s.Path); err != nil {
		return err
	}
	var fileData []byte
	if err := dec.Decode(&fileData); err != nil {
		return err
	}
	fileReader := bytes.NewReader(fileData)
	file, err := typedjson.Decode(fileReader)
	if err != nil {
		return err
	}
	s.File = file.(*syntax.File)
	return nil
}

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
	FindPkgs(ctx context.Context, pkgs []string) (map[string][]db.Package, []string, error)
}

type Config interface {
	GetPaths() *config.Paths
	PagerStyle() string
}

type FunctionsOutput struct {
	Contents *[]string
}

// EXECUTORS

type ScriptResolverExecutor interface {
	ResolveScript(ctx context.Context, pkg *db.Package) *ScriptInfo
}

type ScriptExecutor interface {
	ReadScript(ctx context.Context, scriptPath string) (*ScriptFile, error)
	ExecuteFirstPass(ctx context.Context, input *BuildInput, sf *ScriptFile) (string, []*types.BuildVars, error)
	PrepareDirs(
		ctx context.Context,
		input *BuildInput,
		basePkg string,
	) error
	ExecuteSecondPass(
		ctx context.Context,
		input *BuildInput,
		sf *ScriptFile,
		varsOfPackages []*types.BuildVars,
		repoDeps []string,
		builtDeps []*BuiltDep,
		basePkg string,
	) ([]*BuiltDep, error)
}

type CacheExecutor interface {
	CheckForBuiltPackage(ctx context.Context, input *BuildInput, vars *types.BuildVars) (string, bool, error)
}

type ScriptViewerExecutor interface {
	ViewScript(ctx context.Context, input *BuildInput, sf *ScriptFile, basePkg string) error
}

type CheckerExecutor interface {
	PerformChecks(
		ctx context.Context,
		input *BuildInput,
		vars *types.BuildVars,
	) (bool, error)
}

type InstallerExecutor interface {
	InstallLocal(paths []string, opts *manager.Opts) error
	Install(pkgs []string, opts *manager.Opts) error
	RemoveAlreadyInstalled(pkgs []string) ([]string, error)
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
	// mgr                  manager.Manager
}

type BuildArgs struct {
	Opts       *types.BuildOpts
	Info       *distro.OSRelease
	PkgFormat_ string
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
	Package  *db.Package
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
		script:     scriptInfo.Script,
		repository: scriptInfo.Repository,
		packages:   args.Packages,
		pkgFormat:  args.PkgFormat(),
		opts:       args.Opts,
		info:       args.Info,
	})
}

func (b *Builder) BuildPackageFromScript(
	ctx context.Context,
	args *BuildPackageFromScriptArgs,
) ([]*BuiltDep, error) {
	return b.BuildPackage(ctx, &BuildInput{
		script:     args.Script,
		repository: "default",
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
		return nil, err
	}

	slog.Debug("ExecuteFirstPass")
	basePkg, varsOfPackages, err := b.scriptExecutor.ExecuteFirstPass(ctx, input, sf)
	if err != nil {
		return nil, err
	}

	var builtDeps []*BuiltDep

	if !input.opts.Clean {
		var remainingVars []*types.BuildVars
		for _, vars := range varsOfPackages {
			builtPkgPath, ok, err := b.cacheExecutor.CheckForBuiltPackage(ctx, input, vars)
			if err != nil {
				return nil, err
			}
			if ok {
				builtDeps = append(builtDeps, &BuiltDep{
					Path: builtPkgPath,
				})
			} else {
				remainingVars = append(remainingVars, vars)
			}
		}

		if len(remainingVars) == 0 {
			return builtDeps, nil
		}
	}

	slog.Debug("ViewScript")
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
		buildDepends = append(buildDepends, vars.BuildDepends...)
		optDepends = append(optDepends, vars.OptDepends...)
		depends = append(depends, vars.Depends...)
		sources = append(sources, vars.Sources...)
		checksums = append(checksums, vars.Checksums...)
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
	alrBuildDeps, err := b.installBuildDeps(ctx, input, buildDepends)
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
	for _, d := range depends {
		if _, found := depNames[d]; !found {
			filteredDepends = append(filteredDepends, d)
		}
	}

	slog.Debug("BuildALRDeps")
	newBuiltDeps, repoDeps, err := b.BuildALRDeps(ctx, input, filteredDepends)
	if err != nil {
		return nil, err
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

	return builtDeps, nil
}

type InstallPkgsArgs struct {
	BuildArgs
	AlrPkgs    []db.Package
	NativePkgs []string
}

func (b *Builder) InstallALRPackages(
	ctx context.Context,
	input interface {
		OsInfoProvider
		BuildOptsProvider
		PkgFormatProvider
	},
	alrPkgs []db.Package,
) error {
	for _, pkg := range alrPkgs {
		res, err := b.BuildPackageFromDb(
			ctx,
			&BuildPackageFromDbArgs{
				Package:  &pkg,
				Packages: []string{},
				BuildArgs: BuildArgs{
					Opts:       input.BuildOpts(),
					Info:       input.OSRelease(),
					PkgFormat_: input.PkgFormat(),
				},
			},
		)
		if err != nil {
			return err
		}

		err = b.installerExecutor.InstallLocal(
			GetBuiltPaths(res),
			&manager.Opts{
				NoConfirm: !input.BuildOpts().Interactive,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
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
	if len(depends) > 0 {
		slog.Info(gotext.Get("Installing dependencies"))

		found, notFound, err := b.repos.FindPkgs(ctx, depends) // Поиск зависимостей
		if err != nil {
			return nil, nil, err
		}
		repoDeps = notFound

		// Если для некоторых пакетов есть несколько опций, упрощаем их все в один срез
		pkgs := cliutils.FlattenPkgs(
			ctx,
			found,
			"install",
			input.BuildOpts().Interactive,
		)
		type item struct {
			pkg      *db.Package
			packages []string
		}
		pkgsMap := make(map[string]*item)
		for _, pkg := range pkgs {
			name := pkg.BasePkgName
			if name == "" {
				name = pkg.Name
			}
			if pkgsMap[name] == nil {
				pkgsMap[name] = &item{
					pkg: &pkg,
				}
			}
			pkgsMap[name].packages = append(
				pkgsMap[name].packages,
				pkg.Name,
			)
		}

		for basePkgName := range pkgsMap {
			pkg := pkgsMap[basePkgName].pkg
			res, err := b.BuildPackageFromDb(
				ctx,
				&BuildPackageFromDbArgs{
					Package:  pkg,
					Packages: pkgsMap[basePkgName].packages,
					BuildArgs: BuildArgs{
						Opts:       input.BuildOpts(),
						Info:       input.OSRelease(),
						PkgFormat_: input.PkgFormat(),
					},
				},
			)
			if err != nil {
				return nil, nil, err
			}

			buildDeps = append(buildDeps, res...)
		}
	}

	repoDeps = removeDuplicates(repoDeps)
	buildDeps = removeDuplicates(buildDeps)

	return buildDeps, repoDeps, nil
}

func (i *Builder) installBuildDeps(
	ctx context.Context,
	input interface {
		OsInfoProvider
		BuildOptsProvider
		PkgFormatProvider
	},
	pkgs []string,
) ([]*BuiltDep, error) {
	var builtDeps []*BuiltDep
	if len(pkgs) > 0 {
		deps, err := i.installerExecutor.RemoveAlreadyInstalled(pkgs)
		if err != nil {
			return nil, err
		}

		builtDeps, err = i.InstallPkgs(ctx, input, deps) // Устанавливаем выбранные пакеты
		if err != nil {
			return nil, err
		}
	}
	return builtDeps, nil
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
	optDeps, err := i.installerExecutor.RemoveAlreadyInstalled(pkgs)
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
	builtDeps, repoDeps, err := i.BuildALRDeps(ctx, input, pkgs)
	if err != nil {
		return nil, err
	}

	if len(builtDeps) > 0 {
		err = i.installerExecutor.InstallLocal(GetBuiltPaths(builtDeps), &manager.Opts{
			NoConfirm: !input.BuildOpts().Interactive,
		})
		if err != nil {
			return nil, err
		}
	}

	if len(repoDeps) > 0 {
		err = i.installerExecutor.Install(repoDeps, &manager.Opts{
			NoConfirm: !input.BuildOpts().Interactive,
		})
		if err != nil {
			return nil, err
		}
	}

	return builtDeps, nil
}
