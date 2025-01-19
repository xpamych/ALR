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

	"plemya-x.ru/alr/internal/config"
	database "plemya-x.ru/alr/internal/db"
	"plemya-x.ru/alr/internal/types"
)

type Config interface {
	GetPaths(ctx context.Context) *config.Paths
	Repos(ctx context.Context) []types.Repo
}

type Repos struct {
	cfg Config
	db  *database.Database
}

func New(
	cfg Config,
	db *database.Database,
) *Repos {
	return &Repos{
		cfg,
		db,
	}
}
