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

package build

import (
	"context"
	"testing"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/manager"
)

type TestPackageFinder struct {
	FindPkgsFunc func(ctx context.Context, pkgs []string) (map[string][]db.Package, []string, error)
}

func (pf *TestPackageFinder) FindPkgs(ctx context.Context, pkgs []string) (map[string][]db.Package, []string, error) {
	if pf.FindPkgsFunc != nil {
		return pf.FindPkgsFunc(ctx, pkgs)
	}
	return map[string][]db.Package{}, []string{}, nil
}

type TestManager struct {
	NameFunc          func() string
	FormatFunc        func() string
	ExistsFunc        func() bool
	SetRootCmdFunc    func(cmd string)
	SyncFunc          func(opts *manager.Opts) error
	InstallFunc       func(opts *manager.Opts, pkgs ...string) error
	RemoveFunc        func(opts *manager.Opts, pkgs ...string) error
	UpgradeFunc       func(opts *manager.Opts, pkgs ...string) error
	InstallLocalFunc  func(opts *manager.Opts, files ...string) error
	UpgradeAllFunc    func(opts *manager.Opts) error
	ListInstalledFunc func(opts *manager.Opts) (map[string]string, error)
	IsInstalledFunc   func(pkg string) (bool, error)
}

func (m *TestManager) Name() string {
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return "TestManager"
}

func (m *TestManager) Format() string {
	if m.FormatFunc != nil {
		return m.FormatFunc()
	}
	return "testpkg"
}

func (m *TestManager) Exists() bool {
	if m.ExistsFunc != nil {
		return m.ExistsFunc()
	}
	return true
}

func (m *TestManager) SetRootCmd(cmd string) {
	if m.SetRootCmdFunc != nil {
		m.SetRootCmdFunc(cmd)
	}
}

func (m *TestManager) Sync(opts *manager.Opts) error {
	if m.SyncFunc != nil {
		return m.SyncFunc(opts)
	}
	return nil
}

func (m *TestManager) Install(opts *manager.Opts, pkgs ...string) error {
	if m.InstallFunc != nil {
		return m.InstallFunc(opts, pkgs...)
	}
	return nil
}

func (m *TestManager) Remove(opts *manager.Opts, pkgs ...string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(opts, pkgs...)
	}
	return nil
}

func (m *TestManager) Upgrade(opts *manager.Opts, pkgs ...string) error {
	if m.UpgradeFunc != nil {
		return m.UpgradeFunc(opts, pkgs...)
	}
	return nil
}

func (m *TestManager) InstallLocal(opts *manager.Opts, files ...string) error {
	if m.InstallLocalFunc != nil {
		return m.InstallLocalFunc(opts, files...)
	}
	return nil
}

func (m *TestManager) UpgradeAll(opts *manager.Opts) error {
	if m.UpgradeAllFunc != nil {
		return m.UpgradeAllFunc(opts)
	}
	return nil
}

func (m *TestManager) ListInstalled(opts *manager.Opts) (map[string]string, error) {
	if m.ListInstalledFunc != nil {
		return m.ListInstalledFunc(opts)
	}
	return map[string]string{}, nil
}

func (m *TestManager) IsInstalled(pkg string) (bool, error) {
	if m.IsInstalledFunc != nil {
		return m.IsInstalledFunc(pkg)
	}
	return true, nil
}

// TODO: fix test
func TestInstallBuildDeps(t *testing.T) {
	type testEnv struct {
		pf   PackageFinder
		vars *types.BuildVars
		opts types.BuildOpts

		// Contains pkgs captured by FindPkgsFunc
		// capturedPkgs []string
	}

	type testCase struct {
		Name     string
		Prepare  func() *testEnv
		Expected func(t *testing.T, e *testEnv, res []string, err error)
	}

	for _, tc := range []testCase{
		/*
			{
				Name: "install only needed deps",
				Prepare: func() *testEnv {
					pf := TestPackageFinder{}
					vars := types.BuildVars{}
					m := TestManager{}
					opts := types.BuildOpts{
						Manager:     &m,
						Interactive: false,
					}

					env := &testEnv{
						pf:           &pf,
						vars:         &vars,
						opts:         opts,
						capturedPkgs: []string{},
					}

					pf.FindPkgsFunc = func(ctx context.Context, pkgs []string) (map[string][]db.Package, []string, error) {
						env.capturedPkgs = append(env.capturedPkgs, pkgs...)
						result := make(map[string][]db.Package)
						result["bar"] = []db.Package{{
							Name: "bar-pkg",
						}}
						result["buz"] = []db.Package{{
							Name: "buz-pkg",
						}}

						return result, []string{}, nil
					}

					vars.BuildDepends = []string{
						"foo",
						"bar",
						"buz",
					}
					m.IsInstalledFunc = func(pkg string) (bool, error) {
						if pkg == "foo" {
							return true, nil
						} else {
							return false, nil
						}
					}

					return env
				},
				Expected: func(t *testing.T, e *testEnv, res []string, err error) {
					assert.NoError(t, err)
					assert.Len(t, res, 2)
					assert.ElementsMatch(t, res, []string{"bar-pkg", "buz-pkg"})

					assert.ElementsMatch(t, e.capturedPkgs, []string{"bar", "buz"})
				},
			},
		*/
	} {
		t.Run(tc.Name, func(tt *testing.T) {
			ctx := context.Background()
			env := tc.Prepare()

			result, err := installBuildDeps(
				ctx,
				env.pf,
				env.vars,
				env.opts,
			)

			tc.Expected(tt, env, result, err)
		})
	}
}
