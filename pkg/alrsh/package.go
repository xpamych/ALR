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

package alrsh

import (
	"fmt"
	"reflect"
	"strings"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/decoder"
)

type PackageNames struct {
	BasePkgName string   `sh:"basepkg_name"`
	Names       []string `sh:"name"`
}

func ParseNames(dec *decoder.Decoder) (*PackageNames, error) {
	var pkgs PackageNames
	err := dec.DecodeVars(&pkgs)
	if err != nil {
		return nil, fmt.Errorf("fail parse names: %w", err)
	}
	return &pkgs, nil
}

type Package struct {
	Repository  string `xorm:"pk 'repository'"`
	Name        string `xorm:"pk 'name'"`
	BasePkgName string `xorm:"notnull 'basepkg_name'"`

	Version       string   `sh:"version" xorm:"notnull 'version'"`
	Release       int      `sh:"release" xorm:"notnull 'release'"`
	Epoch         uint     `sh:"epoch" xorm:"'epoch'"`
	Architectures []string `sh:"architectures" xorm:"json 'architectures'"`
	Licenses      []string `sh:"license" xorm:"json 'licenses'"`
	Provides      []string `sh:"provides" xorm:"json 'provides'"`
	Conflicts     []string `sh:"conflicts" xorm:"json 'conflicts'"`
	Replaces      []string `sh:"replaces" xorm:"json 'replaces'"`

	Summary          OverridableField[string]   `sh:"summary" xorm:"'summary'"`
	Description      OverridableField[string]   `sh:"desc" xorm:"'description'"`
	Group            OverridableField[string]   `sh:"group" xorm:"'group_name'"`
	Homepage         OverridableField[string]   `sh:"homepage" xorm:"'homepage'"`
	Maintainer       OverridableField[string]   `sh:"maintainer" xorm:"'maintainer'"`
	Depends          OverridableField[[]string] `sh:"deps" xorm:"'depends'"`
	BuildDepends     OverridableField[[]string] `sh:"build_deps" xorm:"'builddepends'"`
	OptDepends       OverridableField[[]string] `sh:"opt_deps" xorm:"'optdepends'"`
	Sources          OverridableField[[]string] `sh:"sources" xorm:"-"`
	Checksums        OverridableField[[]string] `sh:"checksums" xorm:"-"`
	Backup           OverridableField[[]string] `sh:"backup" xorm:"-"`
	Scripts          OverridableField[Scripts]  `sh:"scripts" xorm:"-"`
	AutoReq          OverridableField[[]string] `sh:"auto_req" xorm:"-"`
	AutoProv         OverridableField[[]string] `sh:"auto_prov" xorm:"-"`
	AutoReqSkipList  OverridableField[[]string] `sh:"auto_req_skiplist" xorm:"-"`
	AutoProvSkipList OverridableField[[]string] `sh:"auto_prov_skiplist" xorm:"-"`

	FireJailed       OverridableField[bool]              `sh:"firejailed" xorm:"-"`
	FireJailProfiles OverridableField[map[string]string] `sh:"firejail_profiles" xorm:"-"`
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

func ResolvePackage(p *Package, overrides []string) {
	val := reflect.ValueOf(p).Elem()
	typ := val.Type()

	for i := range val.NumField() {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanInterface() {
			continue
		}

		if field.Kind() == reflect.Struct && strings.HasPrefix(fieldType.Type.String(), "alrsh.OverridableField") {
			of := field.Addr().Interface()
			if res, ok := of.(interface {
				Resolve([]string)
			}); ok {
				res.Resolve(overrides)
			}
		}
	}
}
