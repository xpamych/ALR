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

package db_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
)

type TestALRConfig struct{}

func (c *TestALRConfig) GetPaths() *config.Paths {
	return &config.Paths{
		DBPath: ":memory:",
	}
}

func prepareDb() *db.Database {
	database := db.New(&TestALRConfig{})
	database.Init(context.Background())
	return database
}

var testPkg = db.Package{
	Name:    "test",
	Version: "0.0.1",
	Release: 1,
	Epoch:   2,
	Description: map[string]string{
		"en": "Test package",
		"ru": "Проверочный пакет",
	},
	Homepage: map[string]string{
		"en": "https://gitea.plemya-x.ru/xpamych/ALR",
	},
	Maintainer: map[string]string{
		"en": "Evgeniy Khramov <xpamych@yandex.ru>",
		"ru": "Евгений Храмов <xpamych@yandex.ru>",
	},
	Architectures: []string{"arm64", "amd64"},
	Licenses:      []string{"GPL-3.0-or-later"},
	Provides:      []string{"test"},
	Conflicts:     []string{"test"},
	Replaces:      []string{"test-old"},
	Depends: map[string][]string{
		"": {"sudo"},
	},
	BuildDepends: map[string][]string{
		"":     {"golang"},
		"arch": {"go"},
	},
	Repository: "default",
}

func TestInit(t *testing.T) {
	ctx := context.Background()
	database := prepareDb()
	defer database.Close()

	ver, ok := database.GetVersion(ctx)
	if !ok {
		t.Errorf("Expected version to be present")
	} else if ver != db.CurrentVersion {
		t.Errorf("Expected version %d, got %d", db.CurrentVersion, ver)
	}
}

func TestInsertPackage(t *testing.T) {
	ctx := context.Background()
	database := prepareDb()
	defer database.Close()

	err := database.InsertPackage(ctx, testPkg)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	pkgs, err := database.GetPkgs(ctx, "name = 'test' AND repository = 'default'")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	if len(pkgs) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(pkgs))
	}

	if !reflect.DeepEqual(testPkg, pkgs[0]) {
		t.Errorf("Expected test package to be the same as database package")
	}
}

func TestGetPkgs(t *testing.T) {
	ctx := context.Background()
	database := prepareDb()
	defer database.Close()

	x1 := testPkg
	x1.Name = "x1"
	x2 := testPkg
	x2.Name = "x2"

	err := database.InsertPackage(ctx, x1)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	err = database.InsertPackage(ctx, x2)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	pkgs, err := database.GetPkgs(ctx, "name LIKE 'x%'")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	for _, dbPkg := range pkgs {
		if !strings.HasPrefix(dbPkg.Name, "x") {
			t.Errorf("Expected package name to start with 'x', got %s", dbPkg.Name)
		}
	}
}

func TestGetPkg(t *testing.T) {
	ctx := context.Background()
	database := prepareDb()
	defer database.Close()

	x1 := testPkg
	x1.Name = "x1"
	x2 := testPkg
	x2.Name = "x2"

	err := database.InsertPackage(ctx, x1)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	err = database.InsertPackage(ctx, x2)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	pkg, err := database.GetPkg("name LIKE 'x%'")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	if pkg.Name != "x1" {
		t.Errorf("Expected x1 package, got %s", pkg.Name)
	}

	if !reflect.DeepEqual(*pkg, x1) {
		t.Errorf("Expected x1 to be %v, got %v", x1, *pkg)
	}
}

func TestDeletePkgs(t *testing.T) {
	ctx := context.Background()
	database := prepareDb()
	defer database.Close()

	x1 := testPkg
	x1.Name = "x1"
	x2 := testPkg
	x2.Name = "x2"

	err := database.InsertPackage(ctx, x1)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	err = database.InsertPackage(ctx, x2)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	err = database.DeletePkgs(ctx, "name = 'x1'")
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}
}

func TestJsonArrayContains(t *testing.T) {
	ctx := context.Background()
	database := prepareDb()
	defer database.Close()

	x1 := testPkg
	x1.Name = "x1"
	x2 := testPkg
	x2.Name = "x2"
	x2.Provides = append(x2.Provides, "x")

	err := database.InsertPackage(ctx, x1)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	err = database.InsertPackage(ctx, x2)
	if err != nil {
		t.Errorf("Expected no error, got %s", err)
	}

	pkgs, err := database.GetPkgs(ctx, "name = 'x2'")
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	if len(pkgs) != 1 || pkgs[0].Name != "x2" {
		t.Errorf("Expected x2 package, got %v", pkgs)
	}

	// Verify the provides field contains 'x'
	found := false
	for _, p := range pkgs[0].Provides {
		if p == "x" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected provides to contain 'x'")
	}
}
