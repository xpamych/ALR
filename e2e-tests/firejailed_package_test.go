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

//go:build e2e

package e2etests_test

import (
	"fmt"
	"testing"

	"github.com/efficientgo/e2e"
)

func TestE2EFirejailedPackage(t *testing.T) {
	dockerMultipleRun(
		t,
		"firejailed-package",
		COMMON_SYSTEMS,
		func(t *testing.T, r e2e.Runnable) {
			defaultPrepare(t, r)
			execShouldNoError(t, r, "alr", "build", "-p", fmt.Sprintf("%s/firejailed-pkg", REPO_NAME_FOR_E2E_TESTS))
			execShouldError(t, r, "alr", "build", "-p", fmt.Sprintf("%s/firejailed-pkg-incorrect", REPO_NAME_FOR_E2E_TESTS))
			execShouldNoError(t, r, "sh", "-c", "dpkg -c *.deb | grep -q '/usr/lib/alr/firejailed/_usr_bin_danger.sh'")
			execShouldNoError(t, r, "sh", "-c", "dpkg -c *.deb | grep -q '/usr/lib/alr/firejailed/_usr_bin_danger.sh.profile'")
		},
	)
}
