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

// Package depver provides parsing and comparison of versioned dependencies
// in PKGBUILD-style format (e.g., "gcc>=5.0", "openssl>=1.1.0").
package depver

import (
	"strings"

	"gitea.plemya-x.ru/xpamych/vercmp"
)

// Operator represents a version comparison operator.
type Operator string

const (
	OpNone Operator = ""   // No version constraint
	OpEq   Operator = "="  // Equal to
	OpGt   Operator = ">"  // Greater than
	OpGe   Operator = ">=" // Greater than or equal to
	OpLt   Operator = "<"  // Less than
	OpLe   Operator = "<=" // Less than or equal to
)

// Dependency represents a package dependency with optional version constraint.
type Dependency struct {
	Name     string   // Package name (e.g., "gcc")
	Operator Operator // Comparison operator (e.g., OpGe for ">=")
	Version  string   // Version string (e.g., "5.0")
}

// operators lists all supported operators in order of decreasing length
// (to ensure ">=" is matched before ">").
var operators = []Operator{OpGe, OpLe, OpGt, OpLt, OpEq}

// Parse parses a dependency string in PKGBUILD format.
// Examples:
//   - "gcc>=5.0" -> Dependency{Name: "gcc", Operator: OpGe, Version: "5.0"}
//   - "openssl" -> Dependency{Name: "openssl", Operator: OpNone, Version: ""}
//   - "cmake>=3.10" -> Dependency{Name: "cmake", Operator: OpGe, Version: "3.10"}
func Parse(dep string) Dependency {
	dep = strings.TrimSpace(dep)
	if dep == "" {
		return Dependency{}
	}

	// Try each operator (longer ones first)
	for _, op := range operators {
		if idx := strings.Index(dep, string(op)); idx > 0 {
			return Dependency{
				Name:     strings.TrimSpace(dep[:idx]),
				Operator: op,
				Version:  strings.TrimSpace(dep[idx+len(op):]),
			}
		}
	}

	// No operator found - just a package name
	return Dependency{
		Name:     dep,
		Operator: OpNone,
		Version:  "",
	}
}

// ParseMultiple parses multiple dependency strings.
func ParseMultiple(deps []string) []Dependency {
	result := make([]Dependency, 0, len(deps))
	for _, dep := range deps {
		if dep != "" {
			result = append(result, Parse(dep))
		}
	}
	return result
}

// String returns the dependency in PKGBUILD format.
func (d Dependency) String() string {
	if d.Operator == OpNone || d.Version == "" {
		return d.Name
	}
	return d.Name + string(d.Operator) + d.Version
}

// Satisfies checks if the given version satisfies the dependency constraint.
// Returns true if:
//   - The dependency has no version constraint (OpNone)
//   - The installed version satisfies the operator/version requirement
func (d Dependency) Satisfies(installedVersion string) bool {
	if d.Operator == OpNone || d.Version == "" {
		return true
	}

	if installedVersion == "" {
		return false
	}

	// vercmp.Compare returns:
	//   -1 if installedVersion < d.Version
	//    0 if installedVersion == d.Version
	//    1 if installedVersion > d.Version
	cmp := vercmp.Compare(installedVersion, d.Version)

	switch d.Operator {
	case OpEq:
		return cmp == 0
	case OpGt:
		return cmp > 0
	case OpGe:
		return cmp >= 0
	case OpLt:
		return cmp < 0
	case OpLe:
		return cmp <= 0
	default:
		return true
	}
}

// HasVersionConstraint returns true if the dependency has a version constraint.
func (d Dependency) HasVersionConstraint() bool {
	return d.Operator != OpNone && d.Version != ""
}
