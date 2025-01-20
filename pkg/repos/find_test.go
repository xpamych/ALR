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

package repos_test

import (
	"reflect"
	"strings"
	"testing"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/repos"
)

func TestFindPkgs(t *testing.T) {
	e := prepare(t)
	defer cleanup(t, e)

	rs := repos.New(
		e.Cfg,
		e.Db,
	)

	err := rs.Pull(e.Ctx, []types.Repo{
		{
			Name: "default",
			URL:  "https://gitea.plemya-x.ru/xpamych/xpamych-alr-repo.git",
		},
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	found, notFound, err := rs.FindPkgs(
		e.Ctx,
		[]string{"alr", "nonexistentpackage1", "nonexistentpackage2"},
	)
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	if !reflect.DeepEqual(notFound, []string{"nonexistentpackage1", "nonexistentpackage2"}) {
		t.Errorf("Expected 'nonexistentpackage{1,2} not to be found")
	}

	if len(found) != 1 {
		t.Errorf("Expected 1 package found, got %d", len(found))
	}

	alrPkgs, ok := found["alr"]
	if !ok {
		t.Fatalf("Expected 'alr' packages to be found")
	}

	if len(alrPkgs) < 2 {
		t.Errorf("Expected two 'alr' packages to be found")
	}

	for i, pkg := range alrPkgs {
		if !strings.HasPrefix(pkg.Name, "alr") {
			t.Errorf("Expected package name of all found packages to start with 'alr', got %s on element %d", pkg.Name, i)
		}
	}
}

func TestFindPkgsEmpty(t *testing.T) {
	e := prepare(t)
	defer cleanup(t, e)

	rs := repos.New(
		e.Cfg,
		e.Db,
	)

	err := e.Db.InsertPackage(e.Ctx, db.Package{
		Name:       "test1",
		Repository: "default",
		Version:    "0.0.1",
		Release:    1,
		Description: db.NewJSON(map[string]string{
			"en": "Test package 1",
			"ru": "Проверочный пакет 1",
		}),
		Provides: db.NewJSON([]string{""}),
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	err = e.Db.InsertPackage(e.Ctx, db.Package{
		Name:       "test2",
		Repository: "default",
		Version:    "0.0.1",
		Release:    1,
		Description: db.NewJSON(map[string]string{
			"en": "Test package 2",
			"ru": "Проверочный пакет 2",
		}),
		Provides: db.NewJSON([]string{"test"}),
	})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	found, notFound, err := rs.FindPkgs(e.Ctx, []string{"test", ""})
	if err != nil {
		t.Fatalf("Expected no error, got %s", err)
	}

	if len(notFound) != 0 {
		t.Errorf("Expected all packages to be found")
	}

	if len(found) != 1 {
		t.Errorf("Expected 1 package found, got %d", len(found))
	}

	testPkgs, ok := found["test"]
	if !ok {
		t.Fatalf("Expected 'test' packages to be found")
	}

	if len(testPkgs) != 1 {
		t.Errorf("Expected one 'test' package to be found, got %d", len(testPkgs))
	}

	if testPkgs[0].Name != "test2" {
		t.Errorf("Expected 'test2' package, got '%s'", testPkgs[0].Name)
	}
}
