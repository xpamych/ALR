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

	"github.com/efficientgo/e2e"
)

func TestE2EIssue78Mirrors(t *testing.T) {
	dockerMultipleRun(
		t,
		"issue-78-mirrors",
		COMMON_SYSTEMS,
		func(t *testing.T, r e2e.Runnable) {
			defaultPrepare(t, r)
			execShouldNoError(t, r, "sudo", "alr", "repo", "mirror", "add", REPO_NAME_FOR_E2E_TESTS, "https://gitea.plemya-x.ru/Maks1mS/repo-for-tests.git")
			execShouldNoError(t, r, "sudo", "alr", "repo", "set-url", REPO_NAME_FOR_E2E_TESTS, "https://example.com")
			execShouldNoError(t, r, "sudo", "alr", "ref")
			execShouldNoError(t, r, "sudo", "alr", "repo", "mirror", "clear", REPO_NAME_FOR_E2E_TESTS)
			execShouldError(t, r, "sudo", "alr", "ref")

			execShouldNoError(t, r, "sudo", "alr", "repo", "mirror", "add", REPO_NAME_FOR_E2E_TESTS, "https://gitea.plemya-x.ru/Maks1mS/repo-for-tests.git")
			execShouldNoError(t, r, "sudo", "alr", "repo", "mirror", "rm", "--partial", REPO_NAME_FOR_E2E_TESTS, "gitea.plemya-x.ru/Maks1mS")
			execShouldError(t, r, "sudo", "alr", "ref")

			execShouldNoError(t, r, "sudo", "alr", "repo", "mirror", "add", REPO_NAME_FOR_E2E_TESTS, "https://gitea.plemya-x.ru/Maks1mS/repo-for-tests.git")
			execShouldNoError(t, r, "sudo", "alr", "repo", "mirror", "rm", REPO_NAME_FOR_E2E_TESTS, "https://gitea.plemya-x.ru/Maks1mS/repo-for-tests.git")
			execShouldError(t, r, "sudo", "alr", "ref")

			execShouldNoError(t, r, "sudo", "alr", "repo", "mirror", "add", REPO_NAME_FOR_E2E_TESTS, "https://gitea.plemya-x.ru/Maks1mS/repo-for-tests.git")
			execShouldNoError(t, r, "sudo", "alr", "repo", "mirror", "rm", REPO_NAME_FOR_E2E_TESTS, "https://gitea.plemya-x.ru/Maks1mS/repo-for-tests.git")
			execShouldError(t, r, "sudo", "alr", "ref")
		},
	)
}
