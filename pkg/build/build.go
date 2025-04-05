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
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/shlex"
	"github.com/goreleaser/nfpm/v2"
	"github.com/leonelquinteros/gotext"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cpu"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/dl"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/dlcache"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/decoder"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/handlers"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/helpers"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
	finddeps "gitea.plemya-x.ru/Plemya-x/ALR/pkg/build/find_deps"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/distro"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/manager"
)

type PackageFinder interface {
	FindPkgs(ctx context.Context, pkgs []string) (map[string][]db.Package, []string, error)
}

type Config interface {
	GetPaths() *config.Paths
	PagerStyle() string
}

type Builder struct {
	ctx    context.Context
	opts   types.BuildOpts
	info   *distro.OSRelease
	repos  PackageFinder
	config Config
}

func NewBuilder(
	ctx context.Context,
	opts types.BuildOpts,
	repos PackageFinder,
	info *distro.OSRelease,
	config Config,
) *Builder {
	return &Builder{
		ctx:    ctx,
		opts:   opts,
		info:   info,
		repos:  repos,
		config: config,
	}
}

func (b *Builder) UpdateOptsFromPkg(pkg *db.Package, packages []string) {
	repodir := b.config.GetPaths().RepoDir
	b.opts.Repository = pkg.Repository
	if pkg.BasePkgName != "" {
		b.opts.Script = filepath.Join(repodir, pkg.Repository, pkg.BasePkgName, "alr.sh")
		b.opts.Packages = packages
	} else {
		b.opts.Script = filepath.Join(repodir, pkg.Repository, pkg.Name, "alr.sh")
	}
}

func (b *Builder) BuildPackage(ctx context.Context) ([]string, []string, error) {
	fl, err := readScript(b.opts.Script)
	if err != nil {
		return nil, nil, err
	}

	// Первый проход предназначен для получения значений переменных и выполняется
	// до отображения скрипта, чтобы предотвратить выполнение вредоносного кода.
	basePkg, varsOfPackages, err := b.executeFirstPass(fl)
	if err != nil {
		return nil, nil, err
	}

	dirs, err := b.getDirs(basePkg)
	if err != nil {
		return nil, nil, err
	}

	builtPaths := make([]string, 0)

	// Если флаг opts.Clean не установлен, и пакет уже собран,
	// возвращаем его, а не собираем заново.
	if !b.opts.Clean {
		var remainingVars []*types.BuildVars
		for _, vars := range varsOfPackages {
			builtPkgPath, ok, err := b.checkForBuiltPackage(
				vars,
				getPkgFormat(b.opts.Manager),
				dirs.BaseDir,
			)
			if err != nil {
				return nil, nil, err
			}

			if ok {
				builtPaths = append(builtPaths, builtPkgPath)
			} else {
				remainingVars = append(remainingVars, vars)
			}
		}

		if len(remainingVars) == 0 {
			return builtPaths, nil, nil
		}
	}

	// Спрашиваем у пользователя, хочет ли он увидеть скрипт сборки.
	err = cliutils.PromptViewScript(
		ctx,
		b.opts.Script,
		basePkg,
		b.config.PagerStyle(),
		b.opts.Interactive,
	)
	if err != nil {
		slog.Error(gotext.Get("Failed to prompt user to view build script"), "err", err)
		os.Exit(1)
	}

	slog.Info(gotext.Get("Building package"), "name", basePkg)

	// Второй проход будет использоваться для выполнения реального кода,
	// поэтому он не ограничен. Скрипт уже был показан
	// пользователю к этому моменту, так что это должно быть безопасно.
	dec, err := b.executeSecondPass(ctx, fl, dirs)
	if err != nil {
		return nil, nil, err
	}

	// Получаем список установленных пакетов в системе
	installed, err := b.opts.Manager.ListInstalled(nil)
	if err != nil {
		return nil, nil, err
	}

	for _, vars := range varsOfPackages {
		cont, err := b.performChecks(ctx, vars, installed) // Выполняем различные проверки
		if err != nil {
			return nil, nil, err
		} else if !cont {
			os.Exit(1) // Если проверки не пройдены, выходим из программы
		}
	}

	// Подготавливаем директории для сборки
	err = prepareDirs(dirs)
	if err != nil {
		return nil, nil, err
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
		os.Exit(1)
	}
	sources, checksums = removeDuplicatesSources(sources, checksums)

	mergedVars := types.BuildVars{
		BuildVarsPre: types.BuildVarsPre{
			Sources:   sources,
			Checksums: checksums,
		},
	}

	buildDeps, err := b.installBuildDeps(ctx, buildDepends) // Устанавливаем зависимости для сборки
	if err != nil {
		return nil, nil, err
	}

	err = b.installOptDeps(ctx, optDepends) // Устанавливаем опциональные зависимости
	if err != nil {
		return nil, nil, err
	}

	newBuildPaths, builtNames, repoDeps, err := b.buildALRDeps(ctx, depends) // Собираем зависимости
	if err != nil {
		return nil, nil, err
	}

	builtPaths = append(builtPaths, newBuildPaths...)

	slog.Info(gotext.Get("Downloading sources")) // Записываем в лог загрузку источников

	err = b.getSources(ctx, dirs, &mergedVars) // Загружаем исходники
	if err != nil {
		return nil, nil, err
	}

	err = b.executeFunctions(ctx, dec, dirs) // Выполняем специальные функции
	if err != nil {
		return nil, nil, err
	}

	for _, vars := range varsOfPackages {
		packageName := ""
		if vars.Base != "" {
			packageName = vars.Name
		}
		funcOut, err := b.executePackageFunctions(ctx, dec, dirs, packageName)
		if err != nil {
			return nil, nil, err
		}

		slog.Info(gotext.Get("Building package metadata"), "name", basePkg)

		pkgFormat := getPkgFormat(b.opts.Manager) // Получаем формат пакета

		pkgInfo, err := b.buildPkgMetadata(ctx, vars, dirs, pkgFormat, append(repoDeps, builtNames...), funcOut.Contents) // Собираем метаданные пакета
		if err != nil {
			return nil, nil, err
		}

		packager, err := nfpm.Get(pkgFormat) // Получаем упаковщик для формата пакета
		if err != nil {
			return nil, nil, err
		}

		pkgName := packager.ConventionalFileName(pkgInfo) // Получаем имя файла пакета
		pkgPath := filepath.Join(dirs.BaseDir, pkgName)   // Определяем путь к пакету

		pkgFile, err := os.Create(pkgPath) // Создаём файл пакета
		if err != nil {
			return nil, nil, err
		}

		slog.Info(gotext.Get("Compressing package"), "name", pkgName) // Логгируем сжатие пакета

		err = packager.Package(pkgInfo, pkgFile) // Упаковываем пакет
		if err != nil {
			return nil, nil, err
		}

		// Добавляем путь и имя только что собранного пакета в
		// соответствующие срезы
		builtPaths = append(builtPaths, pkgPath)
		builtNames = append(builtNames, vars.Name)
	}

	err = b.removeBuildDeps(ctx, buildDeps) // Удаляем зависимости для сборки
	if err != nil {
		return nil, nil, err
	}

	// Удаляем дубликаты из pkgPaths и pkgNames.
	// Дубликаты могут появиться, если несколько зависимостей
	// зависят от одних и тех же пакетов.
	pkgPaths := removeDuplicates(builtPaths)
	pkgNames := removeDuplicates(builtNames)

	return pkgPaths, pkgNames, nil // Возвращаем пути и имена пакетов
}

// Функция executeFirstPass выполняет парсированный скрипт в ограниченной среде,
// чтобы извлечь переменные сборки без выполнения реального кода.
func (b *Builder) executeFirstPass(
	fl *syntax.File,
) (string, []*types.BuildVars, error) {
	varsOfPackages := []*types.BuildVars{}

	scriptDir := filepath.Dir(b.opts.Script)                                   // Получаем директорию скрипта
	env := createBuildEnvVars(b.info, types.Directories{ScriptDir: scriptDir}) // Создаём переменные окружения для сборки

	runner, err := interp.New(
		interp.Env(expand.ListEnviron(env...)),                               // Устанавливаем окружение
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),                         // Устанавливаем стандартный ввод-вывод
		interp.ExecHandler(helpers.Restricted.ExecHandler(handlers.NopExec)), // Ограничиваем выполнение
		interp.ReadDirHandler2(handlers.RestrictedReadDir(scriptDir)),        // Ограничиваем чтение директорий
		interp.StatHandler(handlers.RestrictedStat(scriptDir)),               // Ограничиваем доступ к статистике файлов
		interp.OpenHandler(handlers.RestrictedOpen(scriptDir)),               // Ограничиваем открытие файлов
	)
	if err != nil {
		return "", nil, err
	}

	err = runner.Run(b.ctx, fl) // Запускаем скрипт
	if err != nil {
		return "", nil, err
	}

	dec := decoder.New(b.info, runner) // Создаём новый декодер

	type packages struct {
		BasePkgName string   `sh:"basepkg_name"`
		Names       []string `sh:"name"`
	}

	var pkgs packages
	err = dec.DecodeVars(&pkgs)
	if err != nil {
		return "", nil, err
	}
	if len(pkgs.Names) == 0 {
		return "", nil, errors.New("package name is missing")
	}
	var vars types.BuildVars
	if len(pkgs.Names) == 1 {
		err = dec.DecodeVars(&vars) // Декодируем переменные
		if err != nil {
			return "", nil, err
		}
		varsOfPackages = append(varsOfPackages, &vars)

		return vars.Name, varsOfPackages, nil
	}
	if len(b.opts.Packages) == 0 {
		return "", nil, errors.New("script has multiple packages but package is not specified")
	}

	for _, pkgName := range b.opts.Packages {
		var preVars types.BuildVarsPre
		funcName := fmt.Sprintf("meta_%s", pkgName)
		meta, ok := dec.GetFuncWithSubshell(funcName)
		if !ok {
			return "", nil, errors.New("func is missing")
		}
		r, err := meta(b.ctx)
		if err != nil {
			return "", nil, err
		}
		d := decoder.New(&distro.OSRelease{}, r)
		err = d.DecodeVars(&preVars)
		if err != nil {
			return "", nil, err
		}
		vars := preVars.ToBuildVars()
		vars.Name = pkgName
		vars.Base = pkgs.BasePkgName

		varsOfPackages = append(varsOfPackages, &vars)
	}

	return pkgs.BasePkgName, varsOfPackages, nil // Возвращаем переменные сборки
}

// Функция getDirs возвращает соответствующие директории для скрипта
func (b *Builder) getDirs(basePkg string) (types.Directories, error) {
	scriptPath, err := filepath.Abs(b.opts.Script)
	if err != nil {
		return types.Directories{}, err
	}

	baseDir := filepath.Join(b.config.GetPaths().PkgsDir, basePkg) // Определяем базовую директорию
	return types.Directories{
		BaseDir:   baseDir,
		SrcDir:    filepath.Join(baseDir, "src"),
		PkgDir:    filepath.Join(baseDir, "pkg"),
		ScriptDir: filepath.Dir(scriptPath),
	}, nil
}

// Функция executeSecondPass выполняет скрипт сборки второй раз без каких-либо ограничений. Возвращается декодер,
// который может быть использован для получения функций и переменных из скрипта.
func (b *Builder) executeSecondPass(
	ctx context.Context,
	fl *syntax.File,
	dirs types.Directories,
) (*decoder.Decoder, error) {
	env := createBuildEnvVars(b.info, dirs) // Создаём переменные окружения для сборки

	fakeroot := handlers.FakerootExecHandler(2 * time.Second) // Настраиваем "fakeroot" для выполнения
	runner, err := interp.New(
		interp.Env(expand.ListEnviron(env...)),       // Устанавливаем окружение
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr), // Устанавливаем стандартный ввод-вывод
		interp.ExecHandlers(func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
			return helpers.Helpers.ExecHandler(fakeroot)
		}), // Обрабатываем выполнение через fakeroot
	)
	if err != nil {
		return nil, err
	}

	err = runner.Run(ctx, fl) // Запускаем скрипт
	if err != nil {
		return nil, err
	}

	return decoder.New(b.info, runner), nil // Возвращаем новый декодер
}

// Функция performChecks проверяет различные аспекты в системе, чтобы убедиться, что пакет может быть установлен.
func (b *Builder) performChecks(ctx context.Context, vars *types.BuildVars, installed map[string]string) (bool, error) {
	if !cpu.IsCompatibleWith(cpu.Arch(), vars.Architectures) { // Проверяем совместимость архитектуры
		cont, err := cliutils.YesNoPrompt(
			ctx,
			gotext.Get("Your system's CPU architecture doesn't match this package. Do you want to build anyway?"),
			b.opts.Interactive,
			true,
		)
		if err != nil {
			return false, err
		}

		if !cont {
			return false, nil
		}
	}

	if instVer, ok := installed[vars.Name]; ok { // Если пакет уже установлен, выводим предупреждение
		slog.Warn(gotext.Get("This package is already installed"),
			"name", vars.Name,
			"version", instVer,
		)
	}

	return true, nil
}

// Функция installBuildDeps устанавливает все зависимости сборки, которые еще не установлены, и возвращает
// срез, содержащий имена всех установленных пакетов.
func (b *Builder) installBuildDeps(ctx context.Context, buildDepends []string) ([]string, error) {
	var buildDeps []string
	if len(buildDepends) > 0 {
		deps, err := removeAlreadyInstalled(b.opts, buildDepends)
		if err != nil {
			return nil, err
		}

		found, notFound, err := b.repos.FindPkgs(ctx, deps) // Находим пакеты-зависимости
		if err != nil {
			return nil, err
		}

		slog.Info(gotext.Get("Installing build dependencies")) // Логгируем установку зависимостей

		flattened := cliutils.FlattenPkgs(ctx, found, "install", b.opts.Interactive) // Уплощаем список зависимостей
		buildDeps = packageNames(flattened)
		b.InstallPkgs(ctx, flattened, notFound, b.opts) // Устанавливаем пакеты
	}
	return buildDeps, nil
}

func (b *Builder) getBuildersForPackages(pkgs []db.Package) []*Builder {
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

	builders := []*Builder{}

	for basePkgName := range pkgsMap {
		pkg := pkgsMap[basePkgName].pkg
		builder := *b
		builder.UpdateOptsFromPkg(pkg, pkgsMap[basePkgName].packages)
		builders = append(builders, &builder)
	}

	return builders
}

func (b *Builder) buildALRDeps(ctx context.Context, depends []string) (builtPaths, builtNames, repoDeps []string, err error) {
	if len(depends) > 0 {
		slog.Info(gotext.Get("Installing dependencies"))

		found, notFound, err := b.repos.FindPkgs(ctx, depends) // Поиск зависимостей
		if err != nil {
			return nil, nil, nil, err
		}
		repoDeps = notFound

		// Если для некоторых пакетов есть несколько опций, упрощаем их все в один срез
		pkgs := cliutils.FlattenPkgs(ctx, found, "install", b.opts.Interactive)
		builders := b.getBuildersForPackages(pkgs)
		for _, builder := range builders {
			// Собираем зависимости
			pkgPaths, pkgNames, err := builder.BuildPackage(ctx)
			if err != nil {
				return nil, nil, nil, err
			}

			// Добавляем пути всех собранных пакетов в builtPaths
			builtPaths = append(builtPaths, pkgPaths...)
			// Добавляем пути всех собранных пакетов в builtPaths
			builtNames = append(builtNames, pkgNames...)
		}
	}

	// Удаляем возможные дубликаты, которые могут быть введены, если
	// несколько зависимостей зависят от одних и тех же пакетов.
	repoDeps = removeDuplicates(repoDeps)
	builtPaths = removeDuplicates(builtPaths)
	builtNames = removeDuplicates(builtNames)
	return builtPaths, builtNames, repoDeps, nil
}

func (b *Builder) getSources(ctx context.Context, dirs types.Directories, bv *types.BuildVars) error {
	for i, src := range bv.Sources {
		opts := dl.Options{
			Name:        fmt.Sprintf("%s[%d]", bv.Name, i),
			URL:         src,
			Destination: dirs.SrcDir,
			Progress:    os.Stderr,
			LocalDir:    dirs.ScriptDir,
		}

		if !strings.EqualFold(bv.Checksums[i], "SKIP") {
			// Если контрольная сумма содержит двоеточие, используйте часть до двоеточия
			// как алгоритм, а часть после как фактическую контрольную сумму.
			// В противном случае используйте sha256 по умолчанию с целой строкой как контрольной суммой.
			algo, hashData, ok := strings.Cut(bv.Checksums[i], ":")
			if ok {
				checksum, err := hex.DecodeString(hashData)
				if err != nil {
					return err
				}
				opts.Hash = checksum
				opts.HashAlgorithm = algo
			} else {
				checksum, err := hex.DecodeString(bv.Checksums[i])
				if err != nil {
					return err
				}
				opts.Hash = checksum
			}
		}

		opts.DlCache = dlcache.New(b.config)

		err := dl.Download(ctx, opts)
		if err != nil {
			return err
		}
	}

	return nil
}

// Функция removeBuildDeps спрашивает у пользователя, хочет ли он удалить зависимости,
// установленные для сборки. Если да, использует менеджер пакетов для их удаления.
func (b *Builder) removeBuildDeps(ctx context.Context, buildDeps []string) error {
	if len(buildDeps) > 0 {
		remove, err := cliutils.YesNoPrompt(
			ctx,
			gotext.Get("Would you like to remove the build dependencies?"),
			b.opts.Interactive,
			false,
		)
		if err != nil {
			return err
		}

		if remove {
			err = b.opts.Manager.Remove(
				&manager.Opts{
					AsRoot:    true,
					NoConfirm: true,
				},
				buildDeps...,
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type FunctionsOutput struct {
	Contents *[]string
}

// Функция executeFunctions выполняет специальные функции ALR, такие как version(), prepare() и т.д.
func (b *Builder) executeFunctions(
	ctx context.Context,
	dec *decoder.Decoder,
	dirs types.Directories,
) error {
	/*
		version, ok := dec.GetFunc("version")
		if ok {
			slog.Info(gotext.Get("Executing version()"))

			buf := &bytes.Buffer{}

			err := version(
				ctx,
				interp.Dir(dirs.SrcDir),
				interp.StdIO(os.Stdin, buf, os.Stderr),
			)
			if err != nil {
				return nil, err
			}

			newVer := strings.TrimSpace(buf.String())
			err = setVersion(ctx, dec.Runner, newVer)
			if err != nil {
				return nil, err
			}
			vars.Version = newVer

			slog.Info(gotext.Get("Updating version"), "new", newVer)
		}
	*/

	prepare, ok := dec.GetFunc("prepare")
	if ok {
		slog.Info(gotext.Get("Executing prepare()"))

		err := prepare(ctx, interp.Dir(dirs.SrcDir))
		if err != nil {
			return err
		}
	}

	build, ok := dec.GetFunc("build")
	if ok {
		slog.Info(gotext.Get("Executing build()"))

		err := build(ctx, interp.Dir(dirs.SrcDir))
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Builder) executePackageFunctions(
	ctx context.Context,
	dec *decoder.Decoder,
	dirs types.Directories,
	packageName string,
) (*FunctionsOutput, error) {
	output := &FunctionsOutput{}
	var packageFuncName string
	var filesFuncName string

	if packageName == "" {
		packageFuncName = "package"
		filesFuncName = "files"
	} else {
		packageFuncName = fmt.Sprintf("package_%s", packageName)
		filesFuncName = fmt.Sprintf("files_%s", packageName)
	}
	packageFn, ok := dec.GetFunc(packageFuncName)
	if ok {
		slog.Info(gotext.Get("Executing %s()", packageFuncName))
		err := packageFn(ctx, interp.Dir(dirs.SrcDir))
		if err != nil {
			return nil, err
		}
	}

	files, ok := dec.GetFuncP(filesFuncName, func(ctx context.Context, s *interp.Runner) error {
		// It should be done via interp.RunnerOption,
		// but due to the issues below, it cannot be done.
		// - https://github.com/mvdan/sh/issues/962
		// - https://github.com/mvdan/sh/issues/1125
		script, err := syntax.NewParser().Parse(strings.NewReader("cd $pkgdir && shopt -s globstar"), "")
		if err != nil {
			return err
		}
		return s.Run(ctx, script)
	})

	if ok {
		slog.Info(gotext.Get("Executing %s()", filesFuncName))

		buf := &bytes.Buffer{}

		err := files(
			ctx,
			interp.Dir(dirs.PkgDir),
			interp.StdIO(os.Stdin, buf, os.Stderr),
		)
		if err != nil {
			return nil, err
		}

		contents, err := shlex.Split(buf.String())
		if err != nil {
			return nil, err
		}
		output.Contents = &contents
	}

	return output, nil
}

func (b *Builder) installOptDeps(ctx context.Context, optDepends []string) error {
	optDeps, err := removeAlreadyInstalled(b.opts, optDepends)
	if err != nil {
		return err
	}
	if len(optDeps) > 0 {
		optDeps, err := cliutils.ChooseOptDepends(ctx, optDeps, "install", b.opts.Interactive) // Пользователя просят выбрать опциональные зависимости
		if err != nil {
			return err
		}

		if len(optDeps) == 0 {
			return nil
		}

		found, notFound, err := b.repos.FindPkgs(ctx, optDeps) // Находим опциональные зависимости
		if err != nil {
			return err
		}

		flattened := cliutils.FlattenPkgs(ctx, found, "install", b.opts.Interactive)
		b.InstallPkgs(ctx, flattened, notFound, b.opts) // Устанавливаем выбранные пакеты
	}
	return nil
}

func (b *Builder) InstallPkgs(
	ctx context.Context,
	alrPkgs []db.Package,
	nativePkgs []string,
	opts types.BuildOpts,
) {
	if len(nativePkgs) > 0 {
		err := opts.Manager.Install(nil, nativePkgs...)
		// Если есть нативные пакеты, выполняем их установку
		if err != nil {
			slog.Error(gotext.Get("Error installing native packages"), "err", err)
			os.Exit(1)
			// Логируем и завершаем выполнение при ошибке
		}
	}

	b.InstallALRPackages(ctx, alrPkgs, opts)
	// Устанавливаем скрипты сборки через функцию InstallScripts
}

func (b *Builder) InstallALRPackages(ctx context.Context, pkgs []db.Package, opts types.BuildOpts) {
	builders := b.getBuildersForPackages(pkgs)
	for _, builder := range builders {
		builtPkgs, _, err := builder.BuildPackage(ctx)
		// Выполняем сборку пакета
		if err != nil {
			slog.Error(gotext.Get("Error building package"), "err", err)
			os.Exit(1)
			// Логируем и завершаем выполнение при ошибке сборки
		}

		err = opts.Manager.InstallLocal(nil, builtPkgs...)
		// Устанавливаем локально собранные пакеты
		if err != nil {
			slog.Error(gotext.Get("Error installing package"), "err", err)
			os.Exit(1)
			// Логируем и завершаем выполнение при ошибке установки
		}
	}
}

// Функция buildPkgMetadata создает метаданные для пакета, который будет собран.
func (b *Builder) buildPkgMetadata(
	ctx context.Context,
	vars *types.BuildVars,
	dirs types.Directories,
	pkgFormat string,
	deps []string,
	preferedContents *[]string,
) (*nfpm.Info, error) {
	pkgInfo := getBasePkgInfo(vars, b.info, &b.opts)
	pkgInfo.Description = vars.Description
	pkgInfo.Platform = "linux"
	pkgInfo.Homepage = vars.Homepage
	pkgInfo.License = strings.Join(vars.Licenses, ", ")
	pkgInfo.Maintainer = vars.Maintainer
	pkgInfo.Overridables = nfpm.Overridables{
		Conflicts: append(vars.Conflicts, vars.Name),
		Replaces:  vars.Replaces,
		Provides:  append(vars.Provides, vars.Name),
		Depends:   deps,
	}

	if pkgFormat == "apk" {
		// Alpine отказывается устанавливать пакеты, которые предоставляют сами себя, поэтому удаляем такие элементы
		pkgInfo.Overridables.Provides = slices.DeleteFunc(pkgInfo.Overridables.Provides, func(s string) bool {
			return s == pkgInfo.Name
		})
	}

	if vars.Epoch != 0 {
		pkgInfo.Epoch = strconv.FormatUint(uint64(vars.Epoch), 10)
	}

	setScripts(vars, pkgInfo, dirs.ScriptDir)

	if slices.Contains(vars.Architectures, "all") {
		pkgInfo.Arch = "all"
	}

	contents, err := buildContents(vars, dirs, preferedContents)
	if err != nil {
		return nil, err
	}
	pkgInfo.Overridables.Contents = contents

	if len(vars.AutoProv) == 1 && decoder.IsTruthy(vars.AutoProv[0]) {
		f := finddeps.New(b.info, pkgFormat)
		err = f.FindProvides(ctx, pkgInfo, dirs, vars.AutoProvSkipList)
		if err != nil {
			return nil, err
		}
	}

	if len(vars.AutoReq) == 1 && decoder.IsTruthy(vars.AutoReq[0]) {
		f := finddeps.New(b.info, pkgFormat)
		err = f.FindRequires(ctx, pkgInfo, dirs, vars.AutoReqSkipList)
		if err != nil {
			return nil, err
		}
	}

	return pkgInfo, nil
}

// Функция checkForBuiltPackage пытается обнаружить ранее собранный пакет и вернуть его путь
// и true, если нашла. Если нет, возвратит "", false, nil.
func (b *Builder) checkForBuiltPackage(
	vars *types.BuildVars,
	pkgFormat,
	baseDir string,
) (string, bool, error) {
	filename, err := b.pkgFileName(vars, pkgFormat)
	if err != nil {
		return "", false, err
	}

	pkgPath := filepath.Join(baseDir, filename)

	_, err = os.Stat(pkgPath)
	if err != nil {
		return "", false, nil
	}

	return pkgPath, true, nil
}

// pkgFileName returns the filename of the package if it were to be built.
// This is used to check if the package has already been built.
func (b *Builder) pkgFileName(vars *types.BuildVars, pkgFormat string) (string, error) {
	pkgInfo := getBasePkgInfo(vars, b.info, &b.opts)

	packager, err := nfpm.Get(pkgFormat)
	if err != nil {
		return "", err
	}

	return packager.ConventionalFileName(pkgInfo), nil
}
