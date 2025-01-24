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

package helpers

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/handlers"
)

type testCase struct {
	name           string
	dirsToCreate   []string
	filesToCreate  []string
	expectedOutput []string
	args           string
}

func TestFindFilesDoc(t *testing.T) {
	tests := []testCase{
		{
			name: "All dirs",
			dirsToCreate: []string{
				"usr/share/doc/yandex-browser-stable/subdir",
				"usr/share/doc/firefox",
			},
			filesToCreate: []string{
				"usr/share/doc/yandex-browser-stable/README.md",
				"usr/share/doc/yandex-browser-stable/subdir/nested-file.txt",
				"usr/share/doc/firefox/README.md",
			},
			expectedOutput: []string{
				"./usr/share/doc/yandex-browser-stable",
				"./usr/share/doc/yandex-browser-stable/README.md",
				"./usr/share/doc/yandex-browser-stable/subdir",
				"./usr/share/doc/yandex-browser-stable/subdir/nested-file.txt",
				"./usr/share/doc/firefox",
				"./usr/share/doc/firefox/README.md",
			},
			args: "",
		},
		{
			name: "Only selected dir",
			dirsToCreate: []string{
				"usr/share/doc/yandex-browser-stable/subdir",
				"usr/share/doc/firefox",
				"usr/share/doc/foo/yandex-browser-stable",
			},
			filesToCreate: []string{
				"usr/share/doc/yandex-browser-stable/README.md",
				"usr/share/doc/yandex-browser-stable/subdir/nested-file.txt",
				"usr/share/doc/firefox/README.md",
				"usr/share/doc/firefox/yandex-browser-stable",
				"usr/share/doc/foo/yandex-browser-stable/README.md",
			},
			expectedOutput: []string{
				"./usr/share/doc/yandex-browser-stable",
				"./usr/share/doc/yandex-browser-stable/README.md",
				"./usr/share/doc/yandex-browser-stable/subdir",
				"./usr/share/doc/yandex-browser-stable/subdir/nested-file.txt",
			},
			args: "yandex-browser-stable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "test-files-find-doc")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			for _, dir := range tc.dirsToCreate {
				dirPath := filepath.Join(tempDir, dir)
				err := os.MkdirAll(dirPath, 0o755)
				assert.NoError(t, err)
			}

			for _, file := range tc.filesToCreate {
				filePath := filepath.Join(tempDir, file)
				err := os.WriteFile(filePath, []byte("test content"), 0o644)
				assert.NoError(t, err)
			}

			helpers := handlers.ExecFuncs{
				"files-find-doc": filesFindDocCmd,
			}
			buf := &bytes.Buffer{}
			runner, err := interp.New(
				interp.Dir(tempDir),
				interp.StdIO(os.Stdin, buf, os.Stderr),
				interp.ExecHandler(helpers.ExecHandler(interp.DefaultExecHandler(1000))),
			)
			assert.NoError(t, err)

			scriptContent := `
shopt -s globstar
files-find-doc ` + tc.args

			script, err := syntax.NewParser().Parse(strings.NewReader(scriptContent), "")
			assert.NoError(t, err)

			err = runner.Run(context.Background(), script)
			assert.NoError(t, err)

			contents := strings.Fields(strings.TrimSpace(buf.String()))
			assert.ElementsMatch(t, tc.expectedOutput, contents)
		})
	}
}

func TestFindLang(t *testing.T) {
	tests := []testCase{
		{
			name: "All dirs",
			dirsToCreate: []string{
				"usr/share/locale/ru/LC_MESSAGES",
				"usr/share/locale/tr/LC_MESSAGES",
			},
			filesToCreate: []string{
				"usr/share/locale/ru/LC_MESSAGES/yandex-disk.mo",
				"usr/share/locale/ru/LC_MESSAGES/yandex-disk-indicator.mo",
				"usr/share/locale/tr/LC_MESSAGES/yandex-disk.mo",
			},
			expectedOutput: []string{
				"./usr/share/locale/ru/LC_MESSAGES/yandex-disk.mo",
				"./usr/share/locale/ru/LC_MESSAGES/yandex-disk-indicator.mo",
				"./usr/share/locale/tr/LC_MESSAGES/yandex-disk.mo",
			},
			args: "",
		},
		{
			name: "All dirs",
			dirsToCreate: []string{
				"usr/share/locale/ru/LC_MESSAGES",
				"usr/share/locale/tr/LC_MESSAGES",
			},
			filesToCreate: []string{
				"usr/share/locale/ru/LC_MESSAGES/yandex-disk.mo",
				"usr/share/locale/ru/LC_MESSAGES/yandex-disk-indicator.mo",
				"usr/share/locale/tr/LC_MESSAGES/yandex-disk.mo",
			},
			expectedOutput: []string{
				"./usr/share/locale/ru/LC_MESSAGES/yandex-disk.mo",
				"./usr/share/locale/tr/LC_MESSAGES/yandex-disk.mo",
			},
			args: "yandex-disk",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "test-files-find-lang")
			assert.NoError(t, err)
			defer os.RemoveAll(tempDir)

			for _, dir := range tc.dirsToCreate {
				dirPath := filepath.Join(tempDir, dir)
				err := os.MkdirAll(dirPath, 0o755)
				assert.NoError(t, err)
			}

			for _, file := range tc.filesToCreate {
				filePath := filepath.Join(tempDir, file)
				err := os.WriteFile(filePath, []byte("test content"), 0o644)
				assert.NoError(t, err)
			}

			helpers := handlers.ExecFuncs{
				"files-find-lang": filesFindLangCmd,
			}
			buf := &bytes.Buffer{}
			runner, err := interp.New(
				interp.Dir(tempDir),
				interp.StdIO(os.Stdin, buf, os.Stderr),
				interp.ExecHandler(helpers.ExecHandler(interp.DefaultExecHandler(1000))),
			)
			assert.NoError(t, err)

			scriptContent := `
shopt -s globstar
files-find-lang ` + tc.args

			script, err := syntax.NewParser().Parse(strings.NewReader(scriptContent), "")
			assert.NoError(t, err)

			err = runner.Run(context.Background(), script)
			assert.NoError(t, err)

			contents := strings.Fields(strings.TrimSpace(buf.String()))
			assert.ElementsMatch(t, tc.expectedOutput, contents)
		})
	}
}
