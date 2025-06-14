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

package build

import (
	"context"
	"os"
	"path/filepath"

	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/alrsh"
)

type ScriptResolver struct {
	cfg Config
}

type ScriptInfo struct {
	Script     string
	Repository string
}

func (s *ScriptResolver) ResolveScript(
	ctx context.Context,
	pkg *alrsh.Package,
) *ScriptInfo {
	var repository, script string

	repodir := s.cfg.GetPaths().RepoDir
	repository = pkg.Repository

	// First, we check if there is a root alr.sh in the repository
	rootScriptPath := filepath.Join(repodir, repository, "alr.sh")
	if _, err := os.Stat(rootScriptPath); err == nil {
		// A repository with a single alr.sh at the root
		script = rootScriptPath
	} else {
		// Multi-package repository - we are looking for alr.sh in the subfolder
		var scriptPath string
		if pkg.BasePkgName != "" {
			scriptPath = filepath.Join(repodir, repository, pkg.BasePkgName, "alr.sh")
		} else {
			scriptPath = filepath.Join(repodir, repository, pkg.Name, "alr.sh")
		}
		script = scriptPath
	}

	return &ScriptInfo{
		Repository: repository,
		Script:     script,
	}
}
