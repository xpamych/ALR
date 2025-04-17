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
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/manager"
)

func NewInstaller(mgr manager.Manager) *Installer {
	return &Installer{
		mgr: mgr,
	}
}

type Installer struct{ mgr manager.Manager }

func (i *Installer) InstallLocal(paths []string, opts *manager.Opts) error {
	return i.mgr.InstallLocal(opts, paths...)
}

func (i *Installer) Install(pkgs []string, opts *manager.Opts) error {
	return i.mgr.Install(opts, pkgs...)
}

func (i *Installer) RemoveAlreadyInstalled(pkgs []string) ([]string, error) {
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
