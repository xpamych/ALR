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

package repos

import (
	"context"
	"sync"

	"plemya-x.ru/alr/internal/config"
	"plemya-x.ru/alr/internal/db"
	database "plemya-x.ru/alr/internal/db"
	"plemya-x.ru/alr/internal/types"
)

// Pull pulls the provided repositories. If a repo doesn't exist, it will be cloned
// and its packages will be written to the DB. If it does exist, it will be pulled.
// In this case, only changed packages will be processed if possible.
// If repos is set to nil, the repos in the ALR config will be used.
//
// Deprecated: use struct method
func Pull(ctx context.Context, repos []types.Repo) error {
	return GetInstance(ctx).Pull(ctx, repos)
}

// FindPkgs looks for packages matching the inputs inside the database.
// It returns a map that maps the package name input to any packages found for it.
// It also returns a slice that contains the names of all packages that were not found.
//
// Deprecated: use struct method
func FindPkgs(ctx context.Context, pkgs []string) (map[string][]db.Package, []string, error) {
	return GetInstance(ctx).FindPkgs(ctx, pkgs)
}

// =======================
// FOR LEGACY ONLY
// =======================

var (
	reposInstance *Repos
	alrConfigOnce sync.Once
)

// Deprecated: For legacy only
func GetInstance(ctx context.Context) *Repos {
	alrConfigOnce.Do(func() {
		cfg := config.GetInstance(ctx)
		db := database.GetInstance(ctx)

		reposInstance = New(
			cfg,
			db,
		)
	})

	return reposInstance
}
