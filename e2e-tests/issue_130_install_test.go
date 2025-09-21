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

	"go.alt-gnome.ru/capytest"
)

func TestE2EIssue130Install(t *testing.T) {
	runMatrixSuite(
		t,
		"alr install {repo}/{package}",
		COMMON_SYSTEMS,
		func(t *testing.T, r capytest.Runner) {
			t.Parallel()
			defaultPrepare(t, r)

			r.Command("sudo", "alr", "in", fmt.Sprintf("%s/foo-pkg", REPO_NAME_FOR_E2E_TESTS)).
				ExpectSuccess().
				Run(t)

			r.Command("sudo", "alr", "in", fmt.Sprintf("%s/bar-pkg", "NOT_REPO_NAME_FOR_E2E_TESTS")).
				ExpectFailure().
				Run(t)
		},
	)
	runMatrixSuite(
		t,
		"alr install {package}+{repo}",
		COMMON_SYSTEMS,
		func(t *testing.T, r capytest.Runner) {
			t.Parallel()
			defaultPrepare(t, r)

			r.Command("sudo", "alr", "in", fmt.Sprintf("foo-pkg+%s", REPO_NAME_FOR_E2E_TESTS)).
				ExpectSuccess().
				Run(t)

			r.Command("sudo", "alr", "in", fmt.Sprintf("bar-pkg+%s", "NOT_REPO_NAME_FOR_E2E_TESTS")).
				ExpectFailure().
				Run(t)
		},
	)
}
