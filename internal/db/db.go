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
	"log/slog"

	"github.com/leonelquinteros/gotext"
	_ "modernc.org/sqlite"
	"xorm.io/xorm"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
)

const CurrentVersion = 5

type Package struct {
	BasePkgName   string              `sh:"basepkg_name" xorm:"notnull 'basepkg_name'"`
	Name          string              `sh:"name,required" xorm:"notnull unique(name_repo) 'name'"`
	Version       string              `sh:"version,required" xorm:"notnull 'version'"`
	Release       int                 `sh:"release" xorm:"notnull 'release'"`
	Epoch         uint                `sh:"epoch" xorm:"'epoch'"`
	Summary       map[string]string   `xorm:"json 'summary'"`
	Description   map[string]string   `xorm:"json 'description'"`
	Group         map[string]string   `xorm:"json 'group_name'"`
	Homepage      map[string]string   `xorm:"json 'homepage'"`
	Maintainer    map[string]string   `xorm:"json 'maintainer'"`
	Architectures []string            `sh:"architectures" xorm:"json 'architectures'"`
	Licenses      []string            `sh:"license" xorm:"json 'licenses'"`
	Provides      []string            `sh:"provides" xorm:"json 'provides'"`
	Conflicts     []string            `sh:"conflicts" xorm:"json 'conflicts'"`
	Replaces      []string            `sh:"replaces" xorm:"json 'replaces'"`
	Depends       map[string][]string `xorm:"json 'depends'"`
	BuildDepends  map[string][]string `xorm:"json 'builddepends'"`
	OptDepends    map[string][]string `xorm:"json 'optdepends'"`
	Repository    string              `xorm:"notnull unique(name_repo) 'repository'"`
}

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
	engine, err := xorm.NewEngine("sqlite", dsn)
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
	if err := d.engine.Sync2(new(Package), new(Version)); err != nil {
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
	return d.engine.DropTables(new(Package), new(Version))
}

func (d *Database) InsertPackage(ctx context.Context, pkg Package) error {
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

func (d *Database) GetPkgs(_ context.Context, where string, args ...any) ([]Package, error) {
	var pkgs []Package
	err := d.engine.Where(where, args...).Find(&pkgs)
	return pkgs, err
}

func (d *Database) GetPkg(where string, args ...any) (*Package, error) {
	var pkg Package
	has, err := d.engine.Where(where, args...).Get(&pkg)
	if err != nil || !has {
		return nil, err
	}
	return &pkg, nil
}

func (d *Database) DeletePkgs(_ context.Context, where string, args ...any) error {
	_, err := d.engine.Where(where, args...).Delete(&Package{})
	return err
}

func (d *Database) IsEmpty() bool {
	count, err := d.engine.Count(new(Package))
	return err != nil || count == 0
}

func (d *Database) Close() error {
	return d.engine.Close()
}
