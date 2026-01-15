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

package manager

import (
	"testing"
)

func TestNewZypperReturnsCorrectType(t *testing.T) {
	z := NewZypper()
	if z == nil {
		t.Fatal("NewZypper() returned nil")
	}
	if z.Name() != "zypper" {
		t.Errorf("Expected name 'zypper', got '%s'", z.Name())
	}
	if z.Format() != "rpm" {
		t.Errorf("Expected format 'rpm', got '%s'", z.Format())
	}
}

func TestManagersOrder(t *testing.T) {
	// Проверяем, что APT-RPM идёт раньше APT в списке менеджеров
	aptRpmIndex := -1
	aptIndex := -1

	for i, m := range managers {
		switch m.Name() {
		case "apt-rpm":
			aptRpmIndex = i
		case "apt":
			aptIndex = i
		}
	}

	if aptRpmIndex == -1 {
		t.Fatal("APT-RPM not found in managers list")
	}
	if aptIndex == -1 {
		t.Fatal("APT not found in managers list")
	}
	if aptRpmIndex >= aptIndex {
		t.Errorf("APT-RPM (index %d) should come before APT (index %d)", aptRpmIndex, aptIndex)
	}
}
