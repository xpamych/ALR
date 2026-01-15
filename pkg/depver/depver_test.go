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

package depver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Dependency
	}{
		{
			name:  "simple package name",
			input: "gcc",
			expected: Dependency{
				Name:     "gcc",
				Operator: OpNone,
				Version:  "",
			},
		},
		{
			name:  "greater or equal",
			input: "gcc>=5.0",
			expected: Dependency{
				Name:     "gcc",
				Operator: OpGe,
				Version:  "5.0",
			},
		},
		{
			name:  "less or equal",
			input: "openssl<=1.1.0",
			expected: Dependency{
				Name:     "openssl",
				Operator: OpLe,
				Version:  "1.1.0",
			},
		},
		{
			name:  "greater than",
			input: "cmake>3.10",
			expected: Dependency{
				Name:     "cmake",
				Operator: OpGt,
				Version:  "3.10",
			},
		},
		{
			name:  "less than",
			input: "python<4.0",
			expected: Dependency{
				Name:     "python",
				Operator: OpLt,
				Version:  "4.0",
			},
		},
		{
			name:  "equal",
			input: "nodejs=18.0.0",
			expected: Dependency{
				Name:     "nodejs",
				Operator: OpEq,
				Version:  "18.0.0",
			},
		},
		{
			name:  "with spaces around",
			input: "  gcc>=5.0  ",
			expected: Dependency{
				Name:     "gcc",
				Operator: OpGe,
				Version:  "5.0",
			},
		},
		{
			name:  "complex version",
			input: "glibc>=2.17-326",
			expected: Dependency{
				Name:     "glibc",
				Operator: OpGe,
				Version:  "2.17-326",
			},
		},
		{
			name:     "empty string",
			input:    "",
			expected: Dependency{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseMultiple(t *testing.T) {
	input := []string{"gcc>=5.0", "openssl", "cmake>=3.10", ""}
	expected := []Dependency{
		{Name: "gcc", Operator: OpGe, Version: "5.0"},
		{Name: "openssl", Operator: OpNone, Version: ""},
		{Name: "cmake", Operator: OpGe, Version: "3.10"},
	}

	result := ParseMultiple(input)
	assert.Equal(t, expected, result)
}

func TestDependency_String(t *testing.T) {
	tests := []struct {
		name     string
		dep      Dependency
		expected string
	}{
		{
			name:     "no version",
			dep:      Dependency{Name: "gcc", Operator: OpNone, Version: ""},
			expected: "gcc",
		},
		{
			name:     "with version",
			dep:      Dependency{Name: "gcc", Operator: OpGe, Version: "5.0"},
			expected: "gcc>=5.0",
		},
		{
			name:     "equal operator",
			dep:      Dependency{Name: "python", Operator: OpEq, Version: "3.11"},
			expected: "python=3.11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dep.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDependency_Satisfies(t *testing.T) {
	tests := []struct {
		name             string
		dep              Dependency
		installedVersion string
		expected         bool
	}{
		{
			name:             "no constraint - always satisfied",
			dep:              Dependency{Name: "gcc", Operator: OpNone, Version: ""},
			installedVersion: "5.0",
			expected:         true,
		},
		{
			name:             "ge - satisfied",
			dep:              Dependency{Name: "gcc", Operator: OpGe, Version: "5.0"},
			installedVersion: "5.0",
			expected:         true,
		},
		{
			name:             "ge - greater version satisfied",
			dep:              Dependency{Name: "gcc", Operator: OpGe, Version: "5.0"},
			installedVersion: "6.0",
			expected:         true,
		},
		{
			name:             "ge - not satisfied",
			dep:              Dependency{Name: "gcc", Operator: OpGe, Version: "5.0"},
			installedVersion: "4.9",
			expected:         false,
		},
		{
			name:             "gt - satisfied",
			dep:              Dependency{Name: "gcc", Operator: OpGt, Version: "5.0"},
			installedVersion: "5.1",
			expected:         true,
		},
		{
			name:             "gt - equal not satisfied",
			dep:              Dependency{Name: "gcc", Operator: OpGt, Version: "5.0"},
			installedVersion: "5.0",
			expected:         false,
		},
		{
			name:             "le - satisfied",
			dep:              Dependency{Name: "gcc", Operator: OpLe, Version: "5.0"},
			installedVersion: "5.0",
			expected:         true,
		},
		{
			name:             "le - lesser satisfied",
			dep:              Dependency{Name: "gcc", Operator: OpLe, Version: "5.0"},
			installedVersion: "4.9",
			expected:         true,
		},
		{
			name:             "le - not satisfied",
			dep:              Dependency{Name: "gcc", Operator: OpLe, Version: "5.0"},
			installedVersion: "5.1",
			expected:         false,
		},
		{
			name:             "lt - satisfied",
			dep:              Dependency{Name: "gcc", Operator: OpLt, Version: "5.0"},
			installedVersion: "4.9",
			expected:         true,
		},
		{
			name:             "lt - equal not satisfied",
			dep:              Dependency{Name: "gcc", Operator: OpLt, Version: "5.0"},
			installedVersion: "5.0",
			expected:         false,
		},
		{
			name:             "eq - satisfied",
			dep:              Dependency{Name: "gcc", Operator: OpEq, Version: "5.0"},
			installedVersion: "5.0",
			expected:         true,
		},
		{
			name:             "eq - not satisfied",
			dep:              Dependency{Name: "gcc", Operator: OpEq, Version: "5.0"},
			installedVersion: "5.1",
			expected:         false,
		},
		{
			name:             "empty installed version",
			dep:              Dependency{Name: "gcc", Operator: OpGe, Version: "5.0"},
			installedVersion: "",
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dep.Satisfies(tt.installedVersion)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDependency_ForManager(t *testing.T) {
	dep := Dependency{Name: "gcc", Operator: OpGe, Version: "5.0"}

	tests := []struct {
		manager  string
		expected string
	}{
		{"pacman", "gcc>=5.0"},
		{"apt", "gcc"},
		{"dnf", "gcc >= 5.0"},
		{"yum", "gcc >= 5.0"},
		{"apk", "gcc>=5.0"},
		{"zypper", "gcc >= 5.0"},
		{"apt-rpm", "gcc >= 5.0"},
		{"unknown", "gcc>=5.0"},
	}

	for _, tt := range tests {
		t.Run(tt.manager, func(t *testing.T) {
			result := dep.ForManager(tt.manager)
			assert.Equal(t, tt.expected, result)
		})
	}

	// Test without version constraint
	depNoVersion := Dependency{Name: "gcc", Operator: OpNone, Version: ""}
	for _, tt := range tests {
		t.Run(tt.manager+"_no_version", func(t *testing.T) {
			result := depNoVersion.ForManager(tt.manager)
			assert.Equal(t, "gcc", result)
		})
	}
}

func TestDependency_ForNfpm(t *testing.T) {
	dep := Dependency{Name: "gcc", Operator: OpGe, Version: "5.0"}

	tests := []struct {
		format   string
		expected string
	}{
		{"deb", "gcc (>= 5.0)"},
		{"rpm", "gcc >= 5.0"},
		{"apk", "gcc>=5.0"},
		{"archlinux", "gcc>=5.0"},
		{"unknown", "gcc>=5.0"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := dep.ForNfpm(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasVersionConstraint(t *testing.T) {
	tests := []struct {
		name     string
		dep      Dependency
		expected bool
	}{
		{
			name:     "has constraint",
			dep:      Dependency{Name: "gcc", Operator: OpGe, Version: "5.0"},
			expected: true,
		},
		{
			name:     "no operator",
			dep:      Dependency{Name: "gcc", Operator: OpNone, Version: ""},
			expected: false,
		},
		{
			name:     "operator but no version",
			dep:      Dependency{Name: "gcc", Operator: OpGe, Version: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dep.HasVersionConstraint()
			assert.Equal(t, tt.expected, result)
		})
	}
}
