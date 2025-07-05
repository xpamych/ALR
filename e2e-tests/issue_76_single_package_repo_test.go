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
	"testing"

	"go.alt-gnome.ru/capytest"
)

func Test75SinglePackageRepo(t *testing.T) {
	runMatrixSuite(
		t,
		"issue-76-single-package-repo",
		COMMON_SYSTEMS,
		func(t *testing.T, r capytest.Runner) {
			execShouldNoError(t, r,
				"sudo",
				"alr",
				"repo",
				"add",
				REPO_NAME_FOR_E2E_TESTS,
				"https://gitea.plemya-x.ru/Maks1mS/test-single-package-alr-repo.git",
			)
			execShouldNoError(t, r, "sudo", "alr", "ref")
			execShouldNoError(t, r, "sudo", "alr", "repo", "set-ref", REPO_NAME_FOR_E2E_TESTS, "1075c918be")
			execShouldNoError(t, r, "alr", "fix")
			execShouldNoError(t, r, "sudo", "alr", "in", "test-single-repo")
			execShouldNoError(t, r, "sh", "-c", "alr list -U")
			execShouldNoError(t, r, "sh", "-c", "test $(alr list -U | wc -l) -eq 0 || exit 1")
			execShouldNoError(t, r, "sudo", "alr", "repo", "set-ref", REPO_NAME_FOR_E2E_TESTS, "5e361c50d7")
			execShouldNoError(t, r, "sudo", "alr", "ref")
			execShouldNoError(t, r, "sh", "-c", "test $(alr list -U | wc -l) -eq 1 || exit 1")
			execShouldNoError(t, r, "sudo", "alr", "up")
			execShouldNoError(t, r, "sh", "-c", "test $(alr list -U | wc -l) -eq 0 || exit 1")
		},
	)
}
