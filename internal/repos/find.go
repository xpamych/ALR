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

package repos

import (
	"context"
	"fmt"
	"strings"

	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/alrsh"
)

func (rs *Repos) FindPkgs(ctx context.Context, pkgs []string) (map[string][]alrsh.Package, []string, error) {
	found := make(map[string][]alrsh.Package)
	var notFound []string

	for _, pkgName := range pkgs {
		if pkgName == "" {
			continue
		}

		var result []alrsh.Package
		var err error

		switch {
		case strings.Contains(pkgName, "/"):
			// repo/pkg
			parts := strings.SplitN(pkgName, "/", 2)
			repo := parts[0]
			name := parts[1]
			result, err = rs.db.GetPkgs(ctx, "name = ? AND repository = ?", name, repo)

		case strings.Contains(pkgName, "+"):
			// pkg+repo
			parts := strings.SplitN(pkgName, "+", 2)
			name := parts[0]
			repo := parts[1]
			result, err = rs.db.GetPkgs(ctx, "name = ? AND repository = ?", name, repo)

		default:
			result, err = rs.db.GetPkgs(ctx, "json_array_contains(provides, ?)", pkgName)
			if err != nil {
				return nil, nil, fmt.Errorf("FindPkgs: get by provides: %w", err)
			}

			if len(result) == 0 {
				result, err = rs.db.GetPkgs(ctx, "basepkg_name = ?", pkgName)
				if err != nil {
					return nil, nil, fmt.Errorf("FindPkgs: get by basepkg_name: %w", err)
				}
			}

			if len(result) == 0 {
				result, err = rs.db.GetPkgs(ctx, "name LIKE ?", pkgName)
			}
		}

		if err != nil {
			return nil, nil, fmt.Errorf("FindPkgs: lookup for %q failed: %w", pkgName, err)
		}

		if len(result) == 0 {
			notFound = append(notFound, pkgName)
		} else {
			found[pkgName] = result
		}
	}

	return found, notFound, nil
}
