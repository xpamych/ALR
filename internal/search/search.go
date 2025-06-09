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

package search

import (
	"context"

	"github.com/jmoiron/sqlx"

	database "gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
)

type PackagesProvider interface {
	GetPkgs(ctx context.Context, where string, args ...any) (*sqlx.Rows, error)
}

type Searcher struct {
	pp PackagesProvider
}

func New(pp PackagesProvider) *Searcher {
	return &Searcher{
		pp: pp,
	}
}

func (s *Searcher) Search(
	ctx context.Context,
	opts *SearchOptions,
) ([]database.Package, error) {
	var packages []database.Package

	where, args := opts.WhereClause()
	result, err := s.pp.GetPkgs(ctx, where, args...)
	if err != nil {
		return nil, err
	}

	for result.Next() {
		var dbPkg database.Package
		err = result.StructScan(&dbPkg)
		if err != nil {
			return nil, err
		}
		packages = append(packages, dbPkg)
	}

	return packages, nil
}
