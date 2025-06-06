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

func TestE2EIssue32Interactive(t *testing.T) {
	dockerMultipleRun(
		t,
		"issue-32-interactive",
		COMMON_SYSTEMS,
		func(t *testing.T, r e2e.Runnable) {
			execShouldNoError(t, r, "alr", "--interactive=false", "remove", "ca-certificates")
			execShouldNoError(t, r, "sudo", "alr", "--interactive=false", "remove", "openssl")
			execShouldNoError(t, r, "alr", "fix")
			execShouldNoError(t, r, "sudo", "apt-get", "update")
			execShouldNoError(t, r, "sudo", "alr", "--interactive=false", "install", "ca-certificates")
		},
	)
}
