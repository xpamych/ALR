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
	"os"
	"path/filepath"

	"github.com/goreleaser/nfpm/v2"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
)

type Cache struct {
	cfg Config
}

func (c *Cache) CheckForBuiltPackage(
	ctx context.Context,
	input *BuildInput,
	vars *types.BuildVars,
) (string, bool, error) {
	filename, err := pkgFileName(input, vars)
	if err != nil {
		return "", false, err
	}

	pkgPath := filepath.Join(getBaseDir(c.cfg, vars.Name), filename)

	_, err = os.Stat(pkgPath)
	if err != nil {
		return "", false, nil
	}

	return pkgPath, true, nil
}

func pkgFileName(
	input interface {
		OsInfoProvider
		PkgFormatProvider
		RepositoryProvider
	},
	vars *types.BuildVars,
) (string, error) {
	pkgInfo := getBasePkgInfo(vars, input)

	packager, err := nfpm.Get(input.PkgFormat())
	if err != nil {
		return "", err
	}

	return packager.ConventionalFileName(pkgInfo), nil
}
