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

package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/leonelquinteros/gotext"
	_ "modernc.org/sqlite"
	"xorm.io/xorm"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/alrsh"
)

const CurrentVersion = 5

type Version struct {
	Version int `xorm:"'version'"`
}

type Config interface {
	GetPaths() *config.Paths
}

type Database struct {
	engine *xorm.Engine
	config Config
}

func New(config Config) *Database {
	return &Database{
		config: config,
	}
}

func (d *Database) Connect() error {
	dsn := d.config.GetPaths().DBPath
	
	// Проверяем директорию для БД
	dbDir := filepath.Dir(dsn)
	if _, err := os.Stat(dbDir); err != nil {
		if os.IsNotExist(err) {
			// Директория не существует - не пытаемся создать
			// Пользователь должен использовать alr fix для создания системных каталогов
			return fmt.Errorf("cache directory does not exist, please run 'sudo alr fix' to create it")
		} else {
			return fmt.Errorf("failed to check database directory: %w", err)
		}
	}
	
	engine, err := xorm.NewEngine("sqlite", dsn)
	// engine.SetLogLevel(log.LOG_DEBUG)
	// engine.ShowSQL(true)
	if err != nil {
		return err
	}
	d.engine = engine
	return nil
}

func (d *Database) Init(ctx context.Context) error {
	if err := d.Connect(); err != nil {
		return err
	}
	if err := d.engine.Sync2(new(alrsh.Package), new(Version)); err != nil {
		return err
	}
	ver, ok := d.GetVersion(ctx)
	if ok && ver != CurrentVersion {
		slog.Warn(gotext.Get("Database version mismatch; resetting"), "version", ver, "expected", CurrentVersion)
		if err := d.reset(); err != nil {
			return err
		}
		return d.Init(ctx)
	} else if !ok {
		slog.Warn(gotext.Get("Database version does not exist. Run alr fix if something isn't working."))
		return d.addVersion(CurrentVersion)
	}
	return nil
}

func (d *Database) GetVersion(ctx context.Context) (int, bool) {
	var v Version
	has, err := d.engine.Get(&v)
	if err != nil || !has {
		return 0, false
	}
	return v.Version, true
}

func (d *Database) addVersion(ver int) error {
	_, err := d.engine.Insert(&Version{Version: ver})
	return err
}

func (d *Database) reset() error {
	return d.engine.DropTables(new(alrsh.Package), new(Version))
}

func (d *Database) InsertPackage(ctx context.Context, pkg alrsh.Package) error {
	session := d.engine.Context(ctx)

	affected, err := session.Where("name = ? AND repository = ?", pkg.Name, pkg.Repository).Update(&pkg)
	if err != nil {
		return err
	}

	if affected == 0 {
		_, err = session.Insert(&pkg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Database) GetPkgs(_ context.Context, where string, args ...any) ([]alrsh.Package, error) {
	var pkgs []alrsh.Package
	err := d.engine.Where(where, args...).Find(&pkgs)
	return pkgs, err
}

func (d *Database) GetPkg(where string, args ...any) (*alrsh.Package, error) {
	var pkg alrsh.Package
	has, err := d.engine.Where(where, args...).Get(&pkg)
	if err != nil || !has {
		return nil, err
	}
	return &pkg, nil
}

func (d *Database) DeletePkgs(_ context.Context, where string, args ...any) error {
	_, err := d.engine.Where(where, args...).Delete(&alrsh.Package{})
	return err
}

func (d *Database) IsEmpty() bool {
	count, err := d.engine.Count(new(alrsh.Package))
	return err != nil || count == 0
}

func (d *Database) Close() error {
	return d.engine.Close()
}
