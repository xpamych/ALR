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

package db

import (
	"context"
	"sync"

	"github.com/jmoiron/sqlx"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/loggerctx"
)

// DB returns the ALR database.
// The first time it's called, it opens the SQLite database file.
// Subsequent calls return the same connection.
//
// Deprecated: use struct method
func DB(ctx context.Context) *sqlx.DB {
	return GetInstance(ctx).GetConn()
}

// Close closes the database
//
// Deprecated: use struct method
func Close() error {
	if database != nil {
		return database.Close()
	}
	return nil
}

// IsEmpty returns true if the database has no packages in it, otherwise it returns false.
//
// Deprecated: use struct method
func IsEmpty(ctx context.Context) bool {
	return GetInstance(ctx).IsEmpty(ctx)
}

// InsertPackage adds a package to the database
//
// Deprecated: use struct method
func InsertPackage(ctx context.Context, pkg Package) error {
	return GetInstance(ctx).InsertPackage(ctx, pkg)
}

// GetPkgs returns a result containing packages that match the where conditions
//
// Deprecated: use struct method
func GetPkgs(ctx context.Context, where string, args ...any) (*sqlx.Rows, error) {
	return GetInstance(ctx).GetPkgs(ctx, where, args...)
}

// GetPkg returns a single package that matches the where conditions
//
// Deprecated: use struct method
func GetPkg(ctx context.Context, where string, args ...any) (*Package, error) {
	return GetInstance(ctx).GetPkg(ctx, where, args...)
}

// DeletePkgs deletes all packages matching the where conditions
//
// Deprecated: use struct method
func DeletePkgs(ctx context.Context, where string, args ...any) error {
	return GetInstance(ctx).DeletePkgs(ctx, where, args...)
}

// =======================
// FOR LEGACY ONLY
// =======================

var (
	dbOnce   sync.Once
	database *Database
)

// Deprecated: For legacy only
func GetInstance(ctx context.Context) *Database {
	dbOnce.Do(func() {
		log := loggerctx.From(ctx)
		cfg := config.GetInstance(ctx)
		database = New(cfg)
		err := database.Init(ctx)
		if err != nil {
			log.Fatal("Error opening database").Err(err).Send()
		}
	})
	return database
}
