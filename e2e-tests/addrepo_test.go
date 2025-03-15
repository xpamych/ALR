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
	"regexp"
	"testing"
	"time"

	"github.com/efficientgo/e2e"
	expect "github.com/tailscale/goexpect"
)

func TestE2EAlrAddRepo(t *testing.T) {
	dockerMultipleRun(
		t,
		"add-repo",
		COMMON_SYSTEMS,
		func(t *testing.T, r e2e.Runnable) {
			runTestCommands(t, r, time.Second*10, []expect.Batcher{
				&expect.BSnd{S: "alr addrepo --name alr-repo --url https://gitea.plemya-x.ru/Plemya-x/alr-repo.git ; echo ALR-ADD-REPO-RETURN-CODE $?\n"},
				&expect.BCas{C: []expect.Caser{
					&expect.Case{
						R: regexp.MustCompile(`ALR-ADD-REPO-RETURN-CODE 0\n$`),
						T: expect.OK(),
					},
					&expect.Case{
						R: regexp.MustCompile(`ALR-ADD-REPO-RETURN-CODE \d\n$`),
						T: expect.Fail(expect.NewStatus(expect.Internal, "Unexpected return code!")),
					},
				}},
			})
		},
	)
}
