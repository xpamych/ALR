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

package alrsh

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/cpu"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/decoder"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/handlers"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/helpers"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/distro"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/types"
)

type ALRSh struct {
	file *syntax.File
	path string
}

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

func (s *ALRSh) ParseBuildVars(ctx context.Context, info *distro.OSRelease, packages []string) (string, []*types.BuildVars, error) {
	varsOfPackages := []*types.BuildVars{}

	scriptDir := filepath.Dir(s.path)
	env := createBuildEnvVars(info, types.Directories{ScriptDir: scriptDir})

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

	err = runner.Run(ctx, s.file) // Запускаем скрипт
	if err != nil {
		return "", nil, err
	}

	dec := decoder.New(info, runner) // Создаём новый декодер

	type Packages struct {
		BasePkgName string   `sh:"basepkg_name"`
		Names       []string `sh:"name"`
	}

	var pkgs Packages
	err = dec.DecodeVars(&pkgs)
	if err != nil {
		return "", nil, err
	}

	if len(pkgs.Names) == 0 {
		return "", nil, errors.New("package name is missing")
	}

	var vars types.BuildVars

	if len(pkgs.Names) == 1 {
		err = dec.DecodeVars(&vars)
		if err != nil {
			return "", nil, err
		}
		varsOfPackages = append(varsOfPackages, &vars)

		return vars.Name, varsOfPackages, nil
	}

	var pkgNames []string

	if len(packages) != 0 {
		pkgNames = packages
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

func (a *ALRSh) Path() string {
	return a.path
}

func (a *ALRSh) File() *syntax.File {
	return a.file
}
