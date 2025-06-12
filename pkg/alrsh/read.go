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

package alrsh

import (
	"fmt"
	"io"
	"io/fs"
	"os"

	"mvdan.cc/sh/v3/syntax"
)

type localFs struct{}

func (fs *localFs) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func ReadFromIOReader(r io.Reader, script string) (*ScriptFile, error) {
	file, err := syntax.NewParser().Parse(r, "alr.sh")
	if err != nil {
		return nil, err
	}
	return &ScriptFile{
		file: file,
		path: script,
	}, nil
}

func ReadFromFS(fsys fs.FS, script string) (*ScriptFile, error) {
	fl, err := fsys.Open(script)
	if err != nil {
		return nil, fmt.Errorf("failed to open alr.sh: %w", err)
	}
	defer fl.Close()

	return ReadFromIOReader(fl, script)
}

func ReadFromLocal(script string) (*ScriptFile, error) {
	return ReadFromFS(&localFs{}, script)
}
