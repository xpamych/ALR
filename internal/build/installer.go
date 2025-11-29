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

package build

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/leonelquinteros/gotext"
	"gitea.plemya-x.ru/xpamych/vercmp"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/manager"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/overrides"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/alrsh"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/distro"
)

func NewInstaller(mgr manager.Manager) *Installer {
	return &Installer{
		mgr: mgr,
	}
}

type Installer struct{ mgr manager.Manager }

func (i *Installer) InstallLocal(ctx context.Context, paths []string, opts *manager.Opts) error {
	return i.mgr.InstallLocal(opts, paths...)
}

func (i *Installer) Install(ctx context.Context, pkgs []string, opts *manager.Opts) error {
	return i.mgr.Install(opts, pkgs...)
}

func (i *Installer) Remove(ctx context.Context, pkgs []string, opts *manager.Opts) error {
	return i.mgr.Remove(opts, pkgs...)
}

func (i *Installer) RemoveAlreadyInstalled(ctx context.Context, pkgs []string) ([]string, error) {
	filteredPackages := []string{}

	for _, dep := range pkgs {
		installed, err := i.mgr.IsInstalled(dep)
		if err != nil {
			return nil, err
		}
		if installed {
			continue
		}
		filteredPackages = append(filteredPackages, dep)
	}

	return filteredPackages, nil
}

func (i *Installer) FilterPackagesByVersion(ctx context.Context, packages []alrsh.Package, osRelease *distro.OSRelease) ([]alrsh.Package, error) {
	installedPkgs, err := i.mgr.ListInstalled(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	var filteredPackages []alrsh.Package

	for _, pkg := range packages {
		alrPkgName := fmt.Sprintf("%s+%s", pkg.Name, pkg.Repository)
		installedVer, isInstalled := installedPkgs[alrPkgName]

		if !isInstalled {
			filteredPackages = append(filteredPackages, pkg)
			continue
		}

		repoVer := pkg.Version
		releaseStr := overrides.ReleasePlatformSpecific(pkg.Release, osRelease)

		if pkg.Release != 0 && pkg.Epoch == 0 {
			repoVer = fmt.Sprintf("%s-%s", pkg.Version, releaseStr)
		} else if pkg.Release != 0 && pkg.Epoch != 0 {
			repoVer = fmt.Sprintf("%d:%s-%s", pkg.Epoch, pkg.Version, releaseStr)
		}

		cmp := vercmp.Compare(repoVer, installedVer)

		if cmp > 0 {
			slog.Info(gotext.Get("Package %s is installed with older version %s, will rebuild with version %s", alrPkgName, installedVer, repoVer))
			filteredPackages = append(filteredPackages, pkg)
		} else if cmp == 0 {
			slog.Info(gotext.Get("Package %s is already installed with version %s, skipping build", alrPkgName, installedVer))
		} else {
			slog.Info(gotext.Get("Package %s is installed with newer version %s (repo has %s), skipping build", alrPkgName, installedVer, repoVer))
		}
	}

	return filteredPackages, nil
}
