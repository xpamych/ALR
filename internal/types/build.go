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

type BuildOpts struct {
	Clean       bool
	Interactive bool
}

type BuildVarsPre struct {
	Version          string   `sh:"version,required"`
	Release          int      `sh:"release,required"`
	Epoch            uint     `sh:"epoch"`
	Summary          string   `sh:"summary"`
	Description      string   `sh:"desc"`
	Group            string   `sh:"group"`
	Homepage         string   `sh:"homepage"`
	Maintainer       string   `sh:"maintainer"`
	Architectures    []string `sh:"architectures"`
	Licenses         []string `sh:"license"`
	Provides         []string `sh:"provides"`
	Conflicts        []string `sh:"conflicts"`
	Depends          []string `sh:"deps"`
	BuildDepends     []string `sh:"build_deps"`
	OptDepends       []string `sh:"opt_deps"`
	Replaces         []string `sh:"replaces"`
	Sources          []string `sh:"sources"`
	Checksums        []string `sh:"checksums"`
	Backup           []string `sh:"backup"`
	Scripts          Scripts  `sh:"scripts"`
	AutoReq          []string `sh:"auto_req"`
	AutoProv         []string `sh:"auto_prov"`
	AutoReqSkipList  []string `sh:"auto_req_skiplist"`
	AutoProvSkipList []string `sh:"auto_prov_skiplist"`
}

func (bv *BuildVarsPre) ToBuildVars() BuildVars {
	return BuildVars{
		Name:         "",
		Base:         "",
		BuildVarsPre: *bv,
	}
}

// BuildVars represents the script variables required
// to build a package
type BuildVars struct {
	Name string `sh:"name,required"`
	Base string
	BuildVarsPre
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
