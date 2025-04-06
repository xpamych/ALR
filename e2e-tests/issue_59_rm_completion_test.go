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

//go:build e2e

package e2etests_test

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/efficientgo/e2e"
)

func TestE2EIssue59RmCompletion(t *testing.T) {
	dockerMultipleRun(
		t,
		"issue-59-rm-completion",
		COMMON_SYSTEMS,
		func(t *testing.T, r e2e.Runnable) {
			err := r.Exec(e2e.NewCommand(
				"alr",
				"addrepo",
				"--name",
				"alr-repo",
				"--url",
				"https://gitea.plemya-x.ru/Maks1mS/repo-for-tests.git",
			))
			assert.NoError(t, err)

			err = r.Exec(e2e.NewCommand(
				"alr", "in", "foo-pkg", "bar-pkg",
			))
			assert.NoError(t, err)

			err = r.Exec(e2e.NewCommand("sh", "-c", "alr rm --generate-bash-completion | grep ^foo-pkg$"))
			assert.NoError(t, err)
			err = r.Exec(e2e.NewCommand("sh", "-c", "alr rm --generate-bash-completion | grep ^bar-pkg$"))
			assert.NoError(t, err)
			err = r.Exec(e2e.NewCommand("sh", "-c", "alr rm --generate-bash-completion | grep ^test-autoreq-autoprov$"))
			assert.Error(t, err)
		},
	)
}
