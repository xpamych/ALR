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

	finddeps "gitea.plemya-x.ru/Plemya-x/ALR/internal/build/find_deps"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/distro"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/decoder"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/handlers"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/helpers"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
)

type LocalScriptExecutor struct {
	cfg Config
}

func NewLocalScriptExecutor(cfg Config) *LocalScriptExecutor {
	return &LocalScriptExecutor{
		cfg,
	}
}

func (e *LocalScriptExecutor) ReadScript(ctx context.Context, scriptPath string) (*ScriptFile, error) {
	fl, err := readScript(scriptPath)
	if err != nil {
		return nil, err
	}
	return &ScriptFile{
		Path: scriptPath,
		File: fl,
	}, nil
}

func (e *LocalScriptExecutor) ExecuteFirstPass(ctx context.Context, input *BuildInput, sf *ScriptFile) (string, []*types.BuildVars, error) {
	varsOfPackages := []*types.BuildVars{}

	scriptDir := filepath.Dir(sf.Path)
	env := createBuildEnvVars(input.info, types.Directories{ScriptDir: scriptDir})

	runner, err := interp.New(
		interp.Env(expand.ListEnviron(env...)),                               // Устанавливаем окружение
		interp.StdIO(os.Stdin, os.Stderr, os.Stderr),                         // Устанавливаем стандартный ввод-вывод
		interp.ExecHandler(helpers.Restricted.ExecHandler(handlers.NopExec)), // Ограничиваем выполнение
		interp.ReadDirHandler2(handlers.RestrictedReadDir(scriptDir)),        // Ограничиваем чтение директорий
		interp.StatHandler(handlers.RestrictedStat(scriptDir)),               // Ограничиваем доступ к статистике файлов
		interp.OpenHandler(handlers.RestrictedOpen(scriptDir)),               // Ограничиваем открытие файлов
		interp.Dir(scriptDir),
	)
	if err != nil {
		return "", nil, err
	}

	err = runner.Run(ctx, sf.File) // Запускаем скрипт
	if err != nil {
		return "", nil, err
	}

	dec := decoder.New(input.info, runner) // Создаём новый декодер

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

	var pkgNames []string

	if len(input.packages) != 0 {
		pkgNames = input.packages
	} else {
		pkgNames = pkgs.Names
	}

	for _, pkgName := range pkgNames {
		var preVars types.BuildVarsPre
		funcName := fmt.Sprintf("meta_%s", pkgName)
		meta, ok := dec.GetFuncWithSubshell(funcName)
		if !ok {
			return "", nil, fmt.Errorf("func %s is missing", funcName)
		}
		r, err := meta(ctx)
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

	return pkgs.BasePkgName, varsOfPackages, nil
}

func (e *LocalScriptExecutor) PrepareDirs(
	ctx context.Context,
	input *BuildInput,
	basePkg string,
) error {
	dirs, err := getDirs(
		e.cfg,
		input.script,
		basePkg,
	)
	if err != nil {
		return err
	}

	err = prepareDirs(dirs)
	if err != nil {
		return err
	}

	return nil
}

func (e *LocalScriptExecutor) ExecuteSecondPass(
	ctx context.Context,
	input *BuildInput,
	sf *ScriptFile,
	varsOfPackages []*types.BuildVars,
	repoDeps []string,
	builtDeps []*BuiltDep,
	basePkg string,
) ([]*BuiltDep, error) {
	dirs, err := getDirs(e.cfg, sf.Path, basePkg)
	if err != nil {
		return nil, err
	}
	env := createBuildEnvVars(input.info, dirs)

	fakeroot := handlers.FakerootExecHandler(2 * time.Second)
	runner, err := interp.New(
		interp.Env(expand.ListEnviron(env...)),       // Устанавливаем окружение
		interp.StdIO(os.Stdin, os.Stderr, os.Stderr), // Устанавливаем стандартный ввод-вывод
		interp.ExecHandlers(func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
			return helpers.Helpers.ExecHandler(fakeroot)
		}), // Обрабатываем выполнение через fakeroot
	)
	if err != nil {
		return nil, err
	}

	err = runner.Run(ctx, sf.File)
	if err != nil {
		return nil, err
	}

	dec := decoder.New(input.info, runner)

	// var builtPaths []string

	err = e.ExecuteFunctions(ctx, dirs, dec)
	if err != nil {
		return nil, err
	}

	for _, vars := range varsOfPackages {
		packageName := ""
		if vars.Base != "" {
			packageName = vars.Name
		}

		pkgFormat := input.pkgFormat

		funcOut, err := e.ExecutePackageFunctions(
			ctx,
			dec,
			dirs,
			packageName,
		)
		if err != nil {
			return nil, err
		}

		slog.Info(gotext.Get("Building package metadata"), "name", basePkg)

		pkgInfo, err := buildPkgMetadata(
			ctx,
			input,
			vars,
			dirs,
			append(
				repoDeps,
				GetBuiltName(builtDeps)...,
			),
			funcOut.Contents,
		)
		if err != nil {
			return nil, err
		}

		packager, err := nfpm.Get(pkgFormat) // Получаем упаковщик для формата пакета
		if err != nil {
			return nil, err
		}

		pkgName := packager.ConventionalFileName(pkgInfo) // Получаем имя файла пакета
		pkgPath := filepath.Join(dirs.BaseDir, pkgName)   // Определяем путь к пакету

		pkgFile, err := os.Create(pkgPath)
		if err != nil {
			return nil, err
		}

		err = packager.Package(pkgInfo, pkgFile)
		if err != nil {
			return nil, err
		}

		builtDeps = append(builtDeps, &BuiltDep{
			Name: vars.Name,
			Path: pkgPath,
		})
	}

	return builtDeps, nil
}

func buildPkgMetadata(
	ctx context.Context,
	input interface {
		OsInfoProvider
		BuildOptsProvider
		PkgFormatProvider
		RepositoryProvider
	},
	vars *types.BuildVars,
	dirs types.Directories,
	deps []string,
	preferedContents *[]string,
) (*nfpm.Info, error) {
	pkgInfo := getBasePkgInfo(vars, input)
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
	pkgInfo.Section = vars.Group

	pkgFormat := input.PkgFormat()
	info := input.OSRelease()

	if pkgFormat == "apk" {
		// Alpine отказывается устанавливать пакеты, которые предоставляют сами себя, поэтому удаляем такие элементы
		pkgInfo.Overridables.Provides = slices.DeleteFunc(pkgInfo.Overridables.Provides, func(s string) bool {
			return s == pkgInfo.Name
		})
	}

	if pkgFormat == "rpm" {
		pkgInfo.RPM.Group = vars.Group

		if vars.Summary != "" {
			pkgInfo.RPM.Summary = vars.Summary
		} else {
			lines := strings.SplitN(vars.Description, "\n", 2)
			pkgInfo.RPM.Summary = lines[0]
		}
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
		f := finddeps.New(info, pkgFormat)
		err = f.FindProvides(ctx, pkgInfo, dirs, vars.AutoProvSkipList)
		if err != nil {
			return nil, err
		}
	}

	if len(vars.AutoReq) == 1 && decoder.IsTruthy(vars.AutoReq[0]) {
		f := finddeps.New(info, pkgFormat)
		err = f.FindRequires(ctx, pkgInfo, dirs, vars.AutoReqSkipList)
		if err != nil {
			return nil, err
		}
	}

	return pkgInfo, nil
}

func (e *LocalScriptExecutor) ExecuteFunctions(ctx context.Context, dirs types.Directories, dec *decoder.Decoder) error {
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

func (e *LocalScriptExecutor) ExecutePackageFunctions(
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
