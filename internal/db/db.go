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

package db

import (
	"context"
	"log/slog"

	"github.com/jmoiron/sqlx"
	"github.com/leonelquinteros/gotext"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
)

// CurrentVersion is the current version of the database.
// The database is reset if its version doesn't match this.
const CurrentVersion = 3

// Package is a ALR package's database representation
type Package struct {
	BasePkgName   string                    `sh:"base" db:"basepkg_name"`
	Name          string                    `sh:"name,required" db:"name"`
	Version       string                    `sh:"version,required" db:"version"`
	Release       int                       `sh:"release,required" db:"release"`
	Epoch         uint                      `sh:"epoch" db:"epoch"`
	Description   JSON[map[string]string]   `db:"description"`
	Homepage      JSON[map[string]string]   `db:"homepage"`
	Maintainer    JSON[map[string]string]   `db:"maintainer"`
	Architectures JSON[[]string]            `sh:"architectures" db:"architectures"`
	Licenses      JSON[[]string]            `sh:"license" db:"licenses"`
	Provides      JSON[[]string]            `sh:"provides" db:"provides"`
	Conflicts     JSON[[]string]            `sh:"conflicts" db:"conflicts"`
	Replaces      JSON[[]string]            `sh:"replaces" db:"replaces"`
	Depends       JSON[map[string][]string] `db:"depends"`
	BuildDepends  JSON[map[string][]string] `db:"builddepends"`
	OptDepends    JSON[map[string][]string] `db:"optdepends"`
	Repository    string                    `db:"repository"`
}

type version struct {
	Version int `db:"version"`
}

type Config interface {
	GetPaths(ctx context.Context) *config.Paths
}

type Database struct {
	conn   *sqlx.DB
	config Config
}

func New(config Config) *Database {
	return &Database{
		config: config,
	}
}

func (d *Database) Init(ctx context.Context) error {
	err := d.Connect(ctx)
	if err != nil {
		return err
	}
	return d.initDB(ctx)
}

func (d *Database) Connect(ctx context.Context) error {
	dsn := d.config.GetPaths(ctx).DBPath
	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	d.conn = db
	return nil
}

func (d *Database) GetConn() *sqlx.DB {
	return d.conn
}

func (d *Database) initDB(ctx context.Context) error {
	d.conn = d.conn.Unsafe()
	conn := d.conn
	_, err := conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS pkgs (
			basepkg_name  TEXT NOT NULL,
			name          TEXT NOT NULL,
			repository    TEXT NOT NULL,
			version       TEXT NOT NULL,
			release       INT  NOT NULL,
			epoch         INT,
			description   TEXT CHECK(description = 'null' OR (JSON_VALID(description) AND JSON_TYPE(description) = 'object')),
			homepage      TEXT CHECK(homepage = 'null' OR (JSON_VALID(homepage) AND JSON_TYPE(homepage) = 'object')),
			maintainer    TEXT CHECK(maintainer = 'null' OR (JSON_VALID(maintainer) AND JSON_TYPE(maintainer) = 'object')),
			architectures TEXT CHECK(architectures = 'null' OR (JSON_VALID(architectures) AND JSON_TYPE(architectures) = 'array')),
			licenses      TEXT CHECK(licenses = 'null' OR (JSON_VALID(licenses) AND JSON_TYPE(licenses) = 'array')),
			provides      TEXT CHECK(provides = 'null' OR (JSON_VALID(provides) AND JSON_TYPE(provides) = 'array')),
			conflicts     TEXT CHECK(conflicts = 'null' OR (JSON_VALID(conflicts) AND JSON_TYPE(conflicts) = 'array')),
			replaces      TEXT CHECK(replaces = 'null' OR (JSON_VALID(replaces) AND JSON_TYPE(replaces) = 'array')),
			depends       TEXT CHECK(depends = 'null' OR (JSON_VALID(depends) AND JSON_TYPE(depends) = 'object')),
			builddepends  TEXT CHECK(builddepends = 'null' OR (JSON_VALID(builddepends) AND JSON_TYPE(builddepends) = 'object')),
			optdepends    TEXT CHECK(optdepends = 'null' OR (JSON_VALID(optdepends) AND JSON_TYPE(optdepends) = 'object')),
			UNIQUE(name, repository)
		);

		CREATE TABLE IF NOT EXISTS alr_db_version (
			version INT NOT NULL
		);
	`)
	if err != nil {
		return err
	}

	ver, ok := d.GetVersion(ctx)
	if ok && ver != CurrentVersion {
		slog.Warn(gotext.Get("Database version mismatch; resetting"), "version", ver, "expected", CurrentVersion)
		err = d.reset(ctx)
		if err != nil {
			return err
		}
		return d.initDB(ctx)
	} else if !ok {
		slog.Warn(gotext.Get("Database version does not exist. Run alr fix if something isn't working."), "version", ver, "expected", CurrentVersion)
		return d.addVersion(ctx, CurrentVersion)
	}

	return nil
}

func (d *Database) GetVersion(ctx context.Context) (int, bool) {
	var ver version
	err := d.conn.GetContext(ctx, &ver, "SELECT * FROM alr_db_version LIMIT 1;")
	if err != nil {
		return 0, false
	}
	return ver.Version, true
}

func (d *Database) addVersion(ctx context.Context, ver int) error {
	_, err := d.conn.ExecContext(ctx, `INSERT INTO alr_db_version(version) VALUES (?);`, ver)
	return err
}

func (d *Database) reset(ctx context.Context) error {
	_, err := d.conn.ExecContext(ctx, "DROP TABLE IF EXISTS pkgs;")
	if err != nil {
		return err
	}
	_, err = d.conn.ExecContext(ctx, "DROP TABLE IF EXISTS alr_db_version;")
	return err
}

func (d *Database) GetPkgs(ctx context.Context, where string, args ...any) (*sqlx.Rows, error) {
	stream, err := d.conn.QueryxContext(ctx, "SELECT * FROM pkgs WHERE "+where, args...)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (d *Database) GetPkg(ctx context.Context, where string, args ...any) (*Package, error) {
	out := &Package{}
	err := d.conn.GetContext(ctx, out, "SELECT * FROM pkgs WHERE "+where+" LIMIT 1", args...)
	return out, err
}

func (d *Database) DeletePkgs(ctx context.Context, where string, args ...any) error {
	_, err := d.conn.ExecContext(ctx, "DELETE FROM pkgs WHERE "+where, args...)
	return err
}

func (d *Database) IsEmpty(ctx context.Context) bool {
	var count int
	err := d.conn.GetContext(ctx, &count, "SELECT count(1) FROM pkgs;")
	if err != nil {
		return true
	}
	return count == 0
}

func (d *Database) InsertPackage(ctx context.Context, pkg Package) error {
	_, err := d.conn.NamedExecContext(ctx, `
		INSERT OR REPLACE INTO pkgs (
			basepkg_name,
			name,
			repository,
			version,
			release,
			epoch,
			description,
			homepage,
			maintainer,
			architectures,
			licenses,
			provides,
			conflicts,
			replaces,
			depends,
			builddepends,
			optdepends
		) VALUES (
		 	:basepkg_name,
			:name,
			:repository,
			:version,
			:release,
			:epoch,
			:description,
			:homepage,
			:maintainer,
			:architectures,
			:licenses,
			:provides,
			:conflicts,
			:replaces,
			:depends,
			:builddepends,
			:optdepends
		);
	`, pkg)
	return err
}

func (d *Database) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	} else {
		return nil
	}
}
