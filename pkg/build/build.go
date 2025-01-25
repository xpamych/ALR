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
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	// Импортируем пакеты для поддержки различных форматов пакетов (APK, DEB, RPM и ARCH).
	_ "github.com/goreleaser/nfpm/v2/apk"
	_ "github.com/goreleaser/nfpm/v2/arch"
	_ "github.com/goreleaser/nfpm/v2/deb"
	_ "github.com/goreleaser/nfpm/v2/rpm"
	"github.com/leonelquinteros/gotext"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/files"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cliutils"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cpu"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/dl"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/overrides"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/decoder"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/handlers"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/helpers"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/distro"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/manager"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/repos"
)

// Функция BuildPackage выполняет сборку скрипта по указанному пути. Возвращает два среза.
// Один содержит пути к собранным пакетам, другой - имена собранных пакетов.
func BuildPackage(ctx context.Context, opts types.BuildOpts) ([]string, []string, error) {
	reposInstance := repos.GetInstance(ctx)

	info, err := distro.ParseOSRelease(ctx)
	if err != nil {
		return nil, nil, err
	}

	fl, err := parseScript(info, opts.Script)
	if err != nil {
		return nil, nil, err
	}

	// Первый проход предназначен для получения значений переменных и выполняется
	// до отображения скрипта, чтобы предотвратить выполнение вредоносного кода.
	vars, err := executeFirstPass(ctx, info, fl, opts.Script)
	if err != nil {
		return nil, nil, err
	}

	dirs := getDirs(ctx, vars, opts.Script)

	// Если флаг opts.Clean не установлен, и пакет уже собран,
	// возвращаем его, а не собираем заново.
	if !opts.Clean {
		builtPkgPath, ok, err := checkForBuiltPackage(opts.Manager, vars, getPkgFormat(opts.Manager), dirs.BaseDir)
		if err != nil {
			return nil, nil, err
		}

		if ok {
			return []string{builtPkgPath}, nil, err
		}
	}

	// Спрашиваем у пользователя, хочет ли он увидеть скрипт сборки.
	err = cliutils.PromptViewScript(ctx, opts.Script, vars.Name, config.Config(ctx).PagerStyle, opts.Interactive)
	if err != nil {
		slog.Error(gotext.Get("Failed to prompt user to view build script"), "err", err)
		os.Exit(1)
	}

	slog.Info(gotext.Get("Building package"), "name", vars.Name, "version", vars.Version)

	// Второй проход будет использоваться для выполнения реального кода,
	// поэтому он не ограничен. Скрипт уже был показан
	// пользователю к этому моменту, так что это должно быть безопасно.
	dec, err := executeSecondPass(ctx, info, fl, dirs)
	if err != nil {
		return nil, nil, err
	}

	// Получаем список установленных пакетов в системе
	installed, err := opts.Manager.ListInstalled(nil)
	if err != nil {
		return nil, nil, err
	}

	cont, err := performChecks(ctx, vars, opts.Interactive, installed) // Выполняем различные проверки
	if err != nil {
		return nil, nil, err
	} else if !cont {
		os.Exit(1) // Если проверки не пройдены, выходим из программы
	}

	// Подготавливаем директории для сборки
	err = prepareDirs(dirs)
	if err != nil {
		return nil, nil, err
	}

	buildDeps, err := installBuildDeps(ctx, reposInstance, vars, opts) // Устанавливаем зависимости для сборки
	if err != nil {
		return nil, nil, err
	}

	err = installOptDeps(ctx, reposInstance, vars, opts) // Устанавливаем опциональные зависимости
	if err != nil {
		return nil, nil, err
	}

	builtPaths, builtNames, repoDeps, err := buildALRDeps(ctx, opts, vars) // Собираем зависимости
	if err != nil {
		return nil, nil, err
	}

	slog.Info(gotext.Get("Downloading sources")) // Записываем в лог загрузку источников

	err = getSources(ctx, dirs, vars) // Загружаем исходники
	if err != nil {
		return nil, nil, err
	}

	funcOut, err := executeFunctions(ctx, dec, dirs, vars) // Выполняем специальные функции
	if err != nil {
		return nil, nil, err
	}

	slog.Info(gotext.Get("Building package metadata"), "name", vars.Name)

	pkgFormat := getPkgFormat(opts.Manager) // Получаем формат пакета

	pkgInfo, err := buildPkgMetadata(ctx, vars, dirs, pkgFormat, info, append(repoDeps, builtNames...), funcOut.Contents) // Собираем метаданные пакета
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

	err = removeBuildDeps(ctx, buildDeps, opts) // Удаляем зависимости для сборки
	if err != nil {
		return nil, nil, err
	}

	// Добавляем путь и имя только что собранного пакета в
	// соответствующие срезы
	pkgPaths := append(builtPaths, pkgPath)
	pkgNames := append(builtNames, vars.Name)

	// Удаляем дубликаты из pkgPaths и pkgNames.
	// Дубликаты могут появиться, если несколько зависимостей
	// зависят от одних и тех же пакетов.
	pkgPaths = removeDuplicates(pkgPaths)
	pkgNames = removeDuplicates(pkgNames)

	return pkgPaths, pkgNames, nil // Возвращаем пути и имена пакетов
}

// Функция parseScript анализирует скрипт сборки с использованием встроенной реализации bash
func parseScript(info *distro.OSRelease, script string) (*syntax.File, error) {
	fl, err := os.Open(script) // Открываем файл скрипта
	if err != nil {
		return nil, err
	}
	defer fl.Close() // Закрываем файл после выполнения

	file, err := syntax.NewParser().Parse(fl, "alr.sh") // Парсим скрипт с помощью синтаксического анализатора
	if err != nil {
		return nil, err
	}

	return file, nil // Возвращаем синтаксическое дерево
}

// Функция executeFirstPass выполняет парсированный скрипт в ограниченной среде,
// чтобы извлечь переменные сборки без выполнения реального кода.
func executeFirstPass(ctx context.Context, info *distro.OSRelease, fl *syntax.File, script string) (*types.BuildVars, error) {
	scriptDir := filepath.Dir(script)                                        // Получаем директорию скрипта
	env := createBuildEnvVars(info, types.Directories{ScriptDir: scriptDir}) // Создаём переменные окружения для сборки

	runner, err := interp.New(
		interp.Env(expand.ListEnviron(env...)),                               // Устанавливаем окружение
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),                         // Устанавливаем стандартный ввод-вывод
		interp.ExecHandler(helpers.Restricted.ExecHandler(handlers.NopExec)), // Ограничиваем выполнение
		interp.ReadDirHandler(handlers.RestrictedReadDir(scriptDir)),         // Ограничиваем чтение директорий
		interp.StatHandler(handlers.RestrictedStat(scriptDir)),               // Ограничиваем доступ к статистике файлов
		interp.OpenHandler(handlers.RestrictedOpen(scriptDir)),               // Ограничиваем открытие файлов
	)
	if err != nil {
		return nil, err
	}

	err = runner.Run(ctx, fl) // Запускаем скрипт
	if err != nil {
		return nil, err
	}

	dec := decoder.New(info, runner) // Создаём новый декодер

	var vars types.BuildVars
	err = dec.DecodeVars(&vars) // Декодируем переменные
	if err != nil {
		return nil, err
	}

	return &vars, nil // Возвращаем переменные сборки
}

// Функция getDirs возвращает соответствующие директории для скрипта
func getDirs(ctx context.Context, vars *types.BuildVars, script string) types.Directories {
	baseDir := filepath.Join(config.GetPaths(ctx).PkgsDir, vars.Name) // Определяем базовую директорию
	return types.Directories{
		BaseDir:   baseDir,
		SrcDir:    filepath.Join(baseDir, "src"),
		PkgDir:    filepath.Join(baseDir, "pkg"),
		ScriptDir: filepath.Dir(script),
	}
}

// Функция executeSecondPass выполняет скрипт сборки второй раз без каких-либо ограничений. Возвращается декодер,
// который может быть использован для получения функций и переменных из скрипта.
func executeSecondPass(ctx context.Context, info *distro.OSRelease, fl *syntax.File, dirs types.Directories) (*decoder.Decoder, error) {
	env := createBuildEnvVars(info, dirs) // Создаём переменные окружения для сборки

	fakeroot := handlers.FakerootExecHandler(2 * time.Second) // Настраиваем "fakeroot" для выполнения
	runner, err := interp.New(
		interp.Env(expand.ListEnviron(env...)),                    // Устанавливаем окружение
		interp.StdIO(os.Stdin, os.Stdout, os.Stderr),              // Устанавливаем стандартный ввод-вывод
		interp.ExecHandler(helpers.Helpers.ExecHandler(fakeroot)), // Обрабатываем выполнение через fakeroot
	)
	if err != nil {
		return nil, err
	}

	err = runner.Run(ctx, fl) // Запускаем скрипт
	if err != nil {
		return nil, err
	}

	return decoder.New(info, runner), nil // Возвращаем новый декодер
}

// Функция prepareDirs подготавливает директории для сборки.
func prepareDirs(dirs types.Directories) error {
	err := os.RemoveAll(dirs.BaseDir) // Удаляем базовую директорию, если она существует
	if err != nil {
		return err
	}
	err = os.MkdirAll(dirs.SrcDir, 0o755) // Создаем директорию для источников
	if err != nil {
		return err
	}
	return os.MkdirAll(dirs.PkgDir, 0o755) // Создаем директорию для пакетов
}

// Функция performChecks проверяет различные аспекты в системе, чтобы убедиться, что пакет может быть установлен.
func performChecks(ctx context.Context, vars *types.BuildVars, interactive bool, installed map[string]string) (bool, error) {
	if !cpu.IsCompatibleWith(cpu.Arch(), vars.Architectures) { // Проверяем совместимость архитектуры
		cont, err := cliutils.YesNoPrompt(ctx, gotext.Get("Your system's CPU architecture doesn't match this package. Do you want to build anyway?"), interactive, true)
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

type PackageFinder interface {
	FindPkgs(ctx context.Context, pkgs []string) (map[string][]db.Package, []string, error)
}

// Функция installBuildDeps устанавливает все зависимости сборки, которые еще не установлены, и возвращает
// срез, содержащий имена всех установленных пакетов.
func installBuildDeps(ctx context.Context, repos PackageFinder, vars *types.BuildVars, opts types.BuildOpts) ([]string, error) {
	var buildDeps []string
	if len(vars.BuildDepends) > 0 {
		deps, err := removeAlreadyInstalled(opts, vars.BuildDepends)
		if err != nil {
			return nil, err
		}

		found, notFound, err := repos.FindPkgs(ctx, deps) // Находим пакеты-зависимости
		if err != nil {
			return nil, err
		}

		slog.Info(gotext.Get("Installing build dependencies")) // Логгируем установку зависимостей

		flattened := cliutils.FlattenPkgs(ctx, found, "install", opts.Interactive) // Уплощаем список зависимостей
		buildDeps = packageNames(flattened)
		InstallPkgs(ctx, flattened, notFound, opts) // Устанавливаем пакеты
	}
	return buildDeps, nil
}

// Функция installOptDeps спрашивает у пользователя, какие, если таковые имеются, опциональные зависимости он хочет установить.
// Если пользователь решает установить какие-либо опциональные зависимости, выполняется их установка.
func installOptDeps(ctx context.Context, repos PackageFinder, vars *types.BuildVars, opts types.BuildOpts) error {
	optDeps, err := removeAlreadyInstalled(opts, vars.OptDepends)
	if err != nil {
		return err
	}
	if len(optDeps) > 0 {
		optDeps, err := cliutils.ChooseOptDepends(ctx, optDeps, "install", opts.Interactive) // Пользователя просят выбрать опциональные зависимости
		if err != nil {
			return err
		}

		if len(optDeps) == 0 {
			return nil
		}

		found, notFound, err := repos.FindPkgs(ctx, optDeps) // Находим опциональные зависимости
		if err != nil {
			return err
		}

		flattened := cliutils.FlattenPkgs(ctx, found, "install", opts.Interactive)
		InstallPkgs(ctx, flattened, notFound, opts) // Устанавливаем выбранные пакеты
	}
	return nil
}

// Функция buildALRDeps собирает все ALR зависимости пакета. Возвращает пути и имена
// пакетов, которые она собрала, а также все зависимости, которые не были найдены в ALR репозитории,
// чтобы они могли быть установлены из системных репозиториев.
func buildALRDeps(ctx context.Context, opts types.BuildOpts, vars *types.BuildVars) (builtPaths, builtNames, repoDeps []string, err error) {
	if len(vars.Depends) > 0 {
		slog.Info(gotext.Get("Installing dependencies"))

		found, notFound, err := repos.FindPkgs(ctx, vars.Depends) // Поиск зависимостей
		if err != nil {
			return nil, nil, nil, err
		}
		repoDeps = notFound

		// Если для некоторых пакетов есть несколько опций, упрощаем их все в один срез
		pkgs := cliutils.FlattenPkgs(ctx, found, "install", opts.Interactive)
		scripts := GetScriptPaths(ctx, pkgs)
		for _, script := range scripts {
			newOpts := opts
			newOpts.Script = script

			// Собираем зависимости
			pkgPaths, pkgNames, err := BuildPackage(ctx, newOpts)
			if err != nil {
				return nil, nil, nil, err
			}

			// Добавляем пути всех собранных пакетов в builtPaths
			builtPaths = append(builtPaths, pkgPaths...)
			// Добавляем пути всех собранных пакетов в builtPaths
			builtNames = append(builtNames, pkgNames...)
			// Добавляем имя текущего пакета в builtNames
			builtNames = append(builtNames, filepath.Base(filepath.Dir(script)))
		}
	}

	// Удаляем возможные дубликаты, которые могут быть введены, если
	// несколько зависимостей зависят от одних и тех же пакетов.
	repoDeps = removeDuplicates(repoDeps)
	builtPaths = removeDuplicates(builtPaths)
	builtNames = removeDuplicates(builtNames)
	return builtPaths, builtNames, repoDeps, nil
}

type FunctionsOutput struct {
	Contents *[]string
}

// Функция executeFunctions выполняет специальные функции ALR, такие как version(), prepare() и т.д.
func executeFunctions(ctx context.Context, dec *decoder.Decoder, dirs types.Directories, vars *types.BuildVars) (*FunctionsOutput, error) {
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

	prepare, ok := dec.GetFunc("prepare")
	if ok {
		slog.Info(gotext.Get("Executing prepare()"))

		err := prepare(ctx, interp.Dir(dirs.SrcDir))
		if err != nil {
			return nil, err
		}
	}

	build, ok := dec.GetFunc("build")
	if ok {
		slog.Info(gotext.Get("Executing build()"))

		err := build(ctx, interp.Dir(dirs.SrcDir))
		if err != nil {
			return nil, err
		}
	}

	// Выполнение всех функций, начинающихся с package_
	for {
		packageFn, ok := dec.GetFunc("package")
		if ok {
			slog.Info(gotext.Get("Executing package()"))
			err := packageFn(ctx, interp.Dir(dirs.SrcDir))
			if err != nil {
				return nil, err
			}
		}

		/*
			// Проверка на наличие дополнительных функций package_*
			packageFuncName := "package_"
			if packageFunc, ok := dec.GetFunc(packageFuncName); ok {
				slog.Info("Executing " + packageFuncName)
				err = packageFunc(ctx, interp.Dir(dirs.SrcDir))
				if err != nil {
					return err
				}
			} else {
				break // Если больше нет функций package_*, выходим из цикла
			}
		*/
		break
	}

	output := &FunctionsOutput{}

	files, ok := dec.GetFuncP("files", func(ctx context.Context, s *interp.Runner) error {
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
		slog.Info(gotext.Get("Executing files()"))

		buf := &bytes.Buffer{}

		err := files(
			ctx,
			interp.Dir(dirs.PkgDir),
			interp.StdIO(os.Stdin, buf, os.Stderr),
		)
		if err != nil {
			return nil, err
		}

		contents := strings.Fields(strings.TrimSpace(buf.String()))
		output.Contents = &contents
	}

	return output, nil
}

// Функция buildPkgMetadata создает метаданные для пакета, который будет собран.
func buildPkgMetadata(
	ctx context.Context,
	vars *types.BuildVars,
	dirs types.Directories,
	pkgFormat string,
	info *distro.OSRelease,
	deps []string,
	preferedContents *[]string,
) (*nfpm.Info, error) {
	pkgInfo := getBasePkgInfo(vars)
	pkgInfo.Description = vars.Description
	pkgInfo.Platform = "linux"
	pkgInfo.Homepage = vars.Homepage
	pkgInfo.License = strings.Join(vars.Licenses, ", ")
	pkgInfo.Maintainer = vars.Maintainer
	pkgInfo.Overridables = nfpm.Overridables{
		Conflicts: vars.Conflicts,
		Replaces:  vars.Replaces,
		Provides:  vars.Provides,
		Depends:   deps,
	}

	if pkgFormat == "apk" {
		// Alpine отказывается устанавливать пакеты, которые предоставляют сами себя, поэтому удаляем такие элементы
		pkgInfo.Overridables.Provides = slices.DeleteFunc(pkgInfo.Overridables.Provides, func(s string) bool {
			return s == pkgInfo.Name
		})
	}

	pkgInfo.Release = overrides.ReleasePlatformSpecific(vars.Release, info)

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
		if pkgFormat == "rpm" {
			err = rpmFindProvides(ctx, pkgInfo, dirs)
			if err != nil {
				return nil, err
			}
		} else {
			slog.Info(gotext.Get("AutoProv is not implemented for this package format, so it's skipped"))
		}
	}

	if len(vars.AutoReq) == 1 && decoder.IsTruthy(vars.AutoReq[0]) {
		if pkgFormat == "rpm" {
			err = rpmFindRequires(ctx, pkgInfo, dirs)
			if err != nil {
				return nil, err
			}
		} else {
			slog.Info(gotext.Get("AutoReq is not implemented for this package format, so it's skipped"))
		}
	}

	return pkgInfo, nil
}

// Функция buildContents создает секцию содержимого пакета, которая содержит файлы,
// которые будут включены в конечный пакет.
func buildContents(vars *types.BuildVars, dirs types.Directories, preferedContents *[]string) ([]*files.Content, error) {
	contents := []*files.Content{}

	processPath := func(path, trimmed string, prefered bool) error {
		fi, err := os.Lstat(path)
		if err != nil {
			return err
		}

		if fi.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			if !prefered {
				_, err = f.Readdirnames(1)
				if err != io.EOF {
					return nil
				}
			}

			contents = append(contents, &files.Content{
				Source:      path,
				Destination: trimmed,
				Type:        "dir",
				FileInfo: &files.ContentFileInfo{
					MTime: fi.ModTime(),
				},
			})
			return nil
		}

		if fi.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			link = strings.TrimPrefix(link, dirs.PkgDir)

			contents = append(contents, &files.Content{
				Source:      link,
				Destination: trimmed,
				Type:        "symlink",
				FileInfo: &files.ContentFileInfo{
					MTime: fi.ModTime(),
					Mode:  fi.Mode(),
				},
			})
			return nil
		}

		fileContent := &files.Content{
			Source:      path,
			Destination: trimmed,
			FileInfo: &files.ContentFileInfo{
				MTime: fi.ModTime(),
				Mode:  fi.Mode(),
				Size:  fi.Size(),
			},
		}

		if slices.Contains(vars.Backup, trimmed) {
			fileContent.Type = "config|noreplace"
		}

		contents = append(contents, fileContent)
		return nil
	}

	if preferedContents != nil {
		for _, trimmed := range *preferedContents {
			path := filepath.Join(dirs.PkgDir, trimmed)
			if err := processPath(path, trimmed, true); err != nil {
				return nil, err
			}
		}
	} else {
		err := filepath.Walk(dirs.PkgDir, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			trimmed := strings.TrimPrefix(path, dirs.PkgDir)
			return processPath(path, trimmed, false)
		})
		if err != nil {
			return nil, err
		}
	}

	return contents, nil
}

// Функция removeBuildDeps спрашивает у пользователя, хочет ли он удалить зависимости,
// установленные для сборки. Если да, использует менеджер пакетов для их удаления.
func removeBuildDeps(ctx context.Context, buildDeps []string, opts types.BuildOpts) error {
	if len(buildDeps) > 0 {
		remove, err := cliutils.YesNoPrompt(ctx, gotext.Get("Would you like to remove the build dependencies?"), opts.Interactive, false)
		if err != nil {
			return err
		}

		if remove {
			err = opts.Manager.Remove(
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

// Функция checkForBuiltPackage пытается обнаружить ранее собранный пакет и вернуть его путь
// и true, если нашла. Если нет, возвратит "", false, nil.
func checkForBuiltPackage(mgr manager.Manager, vars *types.BuildVars, pkgFormat, baseDir string) (string, bool, error) {
	filename, err := pkgFileName(vars, pkgFormat)
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

func getBasePkgInfo(vars *types.BuildVars) *nfpm.Info {
	return &nfpm.Info{
		Name:    vars.Name,
		Arch:    cpu.Arch(),
		Version: vars.Version,
		Release: strconv.Itoa(vars.Release),
		Epoch:   strconv.FormatUint(uint64(vars.Epoch), 10),
	}
}

// pkgFileName returns the filename of the package if it were to be built.
// This is used to check if the package has already been built.
func pkgFileName(vars *types.BuildVars, pkgFormat string) (string, error) {
	pkgInfo := getBasePkgInfo(vars)

	packager, err := nfpm.Get(pkgFormat)
	if err != nil {
		return "", err
	}

	return packager.ConventionalFileName(pkgInfo), nil
}

// Функция getPkgFormat возвращает формат пакета из менеджера пакетов,
// или ALR_PKG_FORMAT, если он установлен.
func getPkgFormat(mgr manager.Manager) string {
	pkgFormat := mgr.Format()
	if format, ok := os.LookupEnv("ALR_PKG_FORMAT"); ok {
		pkgFormat = format
	}
	return pkgFormat
}

// Функция createBuildEnvVars создает переменные окружения, которые будут установлены
// в скрипте сборки при его выполнении.
func createBuildEnvVars(info *distro.OSRelease, dirs types.Directories) []string {
	env := os.Environ()

	env = append(
		env,
		"DISTRO_NAME="+info.Name,
		"DISTRO_PRETTY_NAME="+info.PrettyName,
		"DISTRO_ID="+info.ID,
		"DISTRO_VERSION_ID="+info.VersionID,
		"DISTRO_ID_LIKE="+strings.Join(info.Like, " "),
		"ARCH="+cpu.Arch(),
		"NCPU="+strconv.Itoa(runtime.NumCPU()),
	)

	if dirs.ScriptDir != "" {
		env = append(env, "scriptdir="+dirs.ScriptDir)
	}

	if dirs.PkgDir != "" {
		env = append(env, "pkgdir="+dirs.PkgDir)
	}

	if dirs.SrcDir != "" {
		env = append(env, "srcdir="+dirs.SrcDir)
	}

	return env
}

// Функция getSources загружает исходники скрипта.
func getSources(ctx context.Context, dirs types.Directories, bv *types.BuildVars) error {
	if len(bv.Sources) != len(bv.Checksums) {
		slog.Error(gotext.Get("The checksums array must be the same length as sources"))
		os.Exit(1)
	}

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

		err := dl.Download(ctx, opts)
		if err != nil {
			return err
		}
	}

	return nil
}

// Функция setScripts добавляет скрипты-перехватчики к метаданным пакета.
func setScripts(vars *types.BuildVars, info *nfpm.Info, scriptDir string) {
	if vars.Scripts.PreInstall != "" {
		info.Scripts.PreInstall = filepath.Join(scriptDir, vars.Scripts.PreInstall)
	}

	if vars.Scripts.PostInstall != "" {
		info.Scripts.PostInstall = filepath.Join(scriptDir, vars.Scripts.PostInstall)
	}

	if vars.Scripts.PreRemove != "" {
		info.Scripts.PreRemove = filepath.Join(scriptDir, vars.Scripts.PreRemove)
	}

	if vars.Scripts.PostRemove != "" {
		info.Scripts.PostRemove = filepath.Join(scriptDir, vars.Scripts.PostRemove)
	}

	if vars.Scripts.PreUpgrade != "" {
		info.ArchLinux.Scripts.PreUpgrade = filepath.Join(scriptDir, vars.Scripts.PreUpgrade)
		info.APK.Scripts.PreUpgrade = filepath.Join(scriptDir, vars.Scripts.PreUpgrade)
	}

	if vars.Scripts.PostUpgrade != "" {
		info.ArchLinux.Scripts.PostUpgrade = filepath.Join(scriptDir, vars.Scripts.PostUpgrade)
		info.APK.Scripts.PostUpgrade = filepath.Join(scriptDir, vars.Scripts.PostUpgrade)
	}

	if vars.Scripts.PreTrans != "" {
		info.RPM.Scripts.PreTrans = filepath.Join(scriptDir, vars.Scripts.PreTrans)
	}

	if vars.Scripts.PostTrans != "" {
		info.RPM.Scripts.PostTrans = filepath.Join(scriptDir, vars.Scripts.PostTrans)
	}
}

// Функция setVersion изменяет переменную версии в скрипте runner.
// Она используется для установки версии на вывод функции version().
func setVersion(ctx context.Context, r *interp.Runner, to string) error {
	fl, err := syntax.NewParser().Parse(strings.NewReader("version='"+to+"'"), "")
	if err != nil {
		return err
	}
	return r.Run(ctx, fl)
}

// Returns not installed dependencies
func removeAlreadyInstalled(opts types.BuildOpts, dependencies []string) ([]string, error) {
	filteredPackages := []string{}

	for _, dep := range dependencies {
		installed, err := opts.Manager.IsInstalled(dep)
		if err != nil {
			return nil, err
		}
		if installed {
			continue
		}
		filteredPackages = append(filteredPackages, dep)
	}

	return filteredPackages, nil
}

// Функция packageNames возвращает имена всех предоставленных пакетов.
func packageNames(pkgs []db.Package) []string {
	names := make([]string, len(pkgs))
	for i, p := range pkgs {
		names[i] = p.Name
	}
	return names
}

// Функция removeDuplicates убирает любые дубликаты из предоставленного среза.
func removeDuplicates(slice []string) []string {
	seen := map[string]struct{}{}
	result := []string{}

	for _, s := range slice {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			result = append(result, s)
		}
	}

	return result
}
