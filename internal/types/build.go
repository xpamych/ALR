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

package types

import "gitea.plemya-x.ru/Plemya-x/ALR/pkg/manager"

type BuildOpts struct {
	Script      string
	Packages    []string
	Manager     manager.Manager
	Clean       bool
	Interactive bool
}

type BuildVarsPre struct {
	Version       string   `sh:"version,required"`
	Release       int      `sh:"release,required"`
	Epoch         uint     `sh:"epoch"`
	Description   string   `sh:"desc"`
	Homepage      string   `sh:"homepage"`
	Maintainer    string   `sh:"maintainer"`
	Architectures []string `sh:"architectures"`
	Licenses      []string `sh:"license"`
	Provides      []string `sh:"provides"`
	Conflicts     []string `sh:"conflicts"`
	Depends       []string `sh:"deps"`
	BuildDepends  []string `sh:"build_deps"`
	OptDepends    []string `sh:"opt_deps"`
	Replaces      []string `sh:"replaces"`
	Sources       []string `sh:"sources"`
	Checksums     []string `sh:"checksums"`
	Backup        []string `sh:"backup"`
	Scripts       Scripts  `sh:"scripts"`
	AutoReq       []string `sh:"auto_req"`
	AutoProv      []string `sh:"auto_prov"`
}

func (bv *BuildVarsPre) ToBuildVars() BuildVars {
	return BuildVars{
		Name:          "",
		Version:       bv.Version,
		Release:       bv.Release,
		Epoch:         bv.Epoch,
		Description:   bv.Description,
		Homepage:      bv.Homepage,
		Maintainer:    bv.Maintainer,
		Architectures: bv.Architectures,
		Licenses:      bv.Licenses,
		Provides:      bv.Provides,
		Conflicts:     bv.Conflicts,
		Depends:       bv.Depends,
		BuildDepends:  bv.BuildDepends,
		OptDepends:    bv.OptDepends,
		Replaces:      bv.Replaces,
		Sources:       bv.Sources,
		Checksums:     bv.Checksums,
		Backup:        bv.Backup,
		Scripts:       bv.Scripts,
		AutoReq:       bv.AutoReq,
		AutoProv:      bv.AutoProv,
	}
}

// BuildVars represents the script variables required
// to build a package
type BuildVars struct {
	Name          string   `sh:"name,required"`
	Version       string   `sh:"version,required"`
	Release       int      `sh:"release,required"`
	Epoch         uint     `sh:"epoch"`
	Description   string   `sh:"desc"`
	Homepage      string   `sh:"homepage"`
	Maintainer    string   `sh:"maintainer"`
	Architectures []string `sh:"architectures"`
	Licenses      []string `sh:"license"`
	Provides      []string `sh:"provides"`
	Conflicts     []string `sh:"conflicts"`
	Depends       []string `sh:"deps"`
	BuildDepends  []string `sh:"build_deps"`
	OptDepends    []string `sh:"opt_deps"`
	Replaces      []string `sh:"replaces"`
	Sources       []string `sh:"sources"`
	Checksums     []string `sh:"checksums"`
	Backup        []string `sh:"backup"`
	Scripts       Scripts  `sh:"scripts"`
	AutoReq       []string `sh:"auto_req"`
	AutoProv      []string `sh:"auto_prov"`
}

type Scripts struct {
	PreInstall  string `sh:"preinstall"`
	PostInstall string `sh:"postinstall"`
	PreRemove   string `sh:"preremove"`
	PostRemove  string `sh:"postremove"`
	PreUpgrade  string `sh:"preupgrade"`
	PostUpgrade string `sh:"postupgrade"`
	PreTrans    string `sh:"pretrans"`
	PostTrans   string `sh:"posttrans"`
}

type Directories struct {
	BaseDir   string
	SrcDir    string
	PkgDir    string
	ScriptDir string
}
