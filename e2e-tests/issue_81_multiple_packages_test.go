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
	"github.com/stretchr/testify/assert"
)

func TestE2EIssue81MultiplePackages(t *testing.T) {
	dockerMultipleRun(
		t,
		"issue-81-multiple-packages",
		COMMON_SYSTEMS,
		func(t *testing.T, r e2e.Runnable) {
			err := r.Exec(e2e.NewCommand(
				"sudo",
				"alr",
				"addrepo",
				"--name",
				"alr-repo",
				"--url",
				REPO_FOR_E2E_TESTS,
			))
			assert.NoError(t, err)

			err = r.Exec(e2e.NewCommand(
				"sudo", "alr", "ref",
			))
			assert.NoError(t, err)

			err = r.Exec(e2e.NewCommand(
				"sudo", "alr", "in", "first-package-with-dashes",
			))
			assert.NoError(t, err)

			err = r.Exec(e2e.NewCommand("cat", "/opt/first-package"))
			assert.NoError(t, err)
		},
	)
}
