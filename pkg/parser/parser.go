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

package parser

import (
	"fmt"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/decoder"
)

type PackageNames struct {
	BasePkgName string   `sh:"basepkg_name"`
	Names       []string `sh:"name"`
}

func ParseNames(dec *decoder.Decoder) (*PackageNames, error) {
	var pkgs PackageNames
	err := dec.DecodeVars(&pkgs)
	if err != nil {
		return nil, fmt.Errorf("fail parse names: %w", err)
	}

	return &pkgs, nil
}
