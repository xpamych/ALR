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

package config

import (
	"context"
)

// Paths contains various paths used by ALR
type Paths struct {
	ConfigDir  string
	ConfigPath string
	CacheDir   string
	RepoDir    string
	PkgsDir    string
	DBPath     string
}

// GetPaths returns a Paths struct.
// The first time it's called, it'll generate the struct
// using information from the system.
// Subsequent calls will return the same value.
//
// Depreacted: use struct API
func GetPaths(ctx context.Context) *Paths {
	alrConfig := GetInstance(ctx)
	return alrConfig.GetPaths(ctx)
}
