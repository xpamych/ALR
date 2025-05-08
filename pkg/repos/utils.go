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

package repos

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/client"

	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/db"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/decoder"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/distro"
)

// isValid makes sure the path of the file being updated is valid.
// It checks to make sure the file is not within a nested directory
// and that it is called alr.sh.
func isValid(from, to diff.File) bool {
	var path string
	if from != nil {
		path = from.Path()
	}
	if to != nil {
		path = to.Path()
	}

	match, _ := filepath.Match("*/*.sh", path)
	return match
}

func parseScript(ctx context.Context, parser *syntax.Parser, runner *interp.Runner, r io.ReadCloser, pkg *db.Package) error {
	defer r.Close()
	fl, err := parser.Parse(r, "alr.sh")
	if err != nil {
		return err
	}

	runner.Reset()
	err = runner.Run(ctx, fl)
	if err != nil {
		return err
	}

	d := decoder.New(&distro.OSRelease{}, runner)
	d.Overrides = false
	d.LikeDistros = false
	return d.DecodeVars(pkg)
}

type PackageInfo struct {
	Version       string            `sh:"version,required"`
	Release       int               `sh:"release,required"`
	Epoch         uint              `sh:"epoch"`
	Architectures db.JSON[[]string] `sh:"architectures"`
	Licenses      db.JSON[[]string] `sh:"license"`
	Provides      db.JSON[[]string] `sh:"provides"`
	Conflicts     db.JSON[[]string] `sh:"conflicts"`
	Replaces      db.JSON[[]string] `sh:"replaces"`
}

func (inf *PackageInfo) ToPackage(repoName string) *db.Package {
	pkg := EmptyPackage(repoName)
	pkg.Version = inf.Version
	pkg.Release = inf.Release
	pkg.Epoch = inf.Epoch
	pkg.Architectures = inf.Architectures
	pkg.Licenses = inf.Licenses
	pkg.Provides = inf.Provides
	pkg.Conflicts = inf.Conflicts
	pkg.Replaces = inf.Replaces
	return pkg
}

func EmptyPackage(repoName string) *db.Package {
	return &db.Package{
		Group:        db.NewJSON(map[string]string{}),
		Summary:      db.NewJSON(map[string]string{}),
		Description:  db.NewJSON(map[string]string{}),
		Homepage:     db.NewJSON(map[string]string{}),
		Maintainer:   db.NewJSON(map[string]string{}),
		Depends:      db.NewJSON(map[string][]string{}),
		BuildDepends: db.NewJSON(map[string][]string{}),
		Repository:   repoName,
	}
}

var overridable = map[string]string{
	"deps":       "Depends",
	"build_deps": "BuildDepends",
	"desc":       "Description",
	"homepage":   "Homepage",
	"maintainer": "Maintainer",
	"group":      "Group",
	"summary":    "Summary",
}

func resolveOverrides(runner *interp.Runner, pkg *db.Package) {
	pkgVal := reflect.ValueOf(pkg).Elem()
	for name, val := range runner.Vars {
		for prefix, field := range overridable {
			if strings.HasPrefix(name, prefix) {
				override := strings.TrimPrefix(name, prefix)
				override = strings.TrimPrefix(override, "_")

				field := pkgVal.FieldByName(field)
				varVal := field.FieldByName("Val")
				varType := varVal.Type()

				switch varType.Elem().String() {
				case "[]string":
					varVal.SetMapIndex(reflect.ValueOf(override), reflect.ValueOf(val.List))
				case "string":
					varVal.SetMapIndex(reflect.ValueOf(override), reflect.ValueOf(val.Str))
				}
				break
			}
		}
	}
}

func getHeadReference(r *git.Repository) (plumbing.ReferenceName, error) {
	remote, err := r.Remote(git.DefaultRemoteName)
	if err != nil {
		return "", err
	}

	endpoint, err := transport.NewEndpoint(remote.Config().URLs[0])
	if err != nil {
		return "", err
	}

	gitClient, err := client.NewClient(endpoint)
	if err != nil {
		return "", err
	}

	session, err := gitClient.NewUploadPackSession(endpoint, nil)
	if err != nil {
		return "", err
	}

	info, err := session.AdvertisedReferences()
	if err != nil {
		return "", err
	}

	refs, err := info.AllReferences()
	if err != nil {
		return "", err
	}

	return refs["HEAD"].Target(), nil
}

func resolveHash(r *git.Repository, ref string) (*plumbing.Hash, error) {
	var err error

	if ref == "" {
		reference, err := getHeadReference(r)
		if err != nil {
			return nil, fmt.Errorf("failed to get head reference %w", err)
		}
		ref = reference.Short()
	}

	hsh, err := r.ResolveRevision(git.DefaultRemoteName + "/" + plumbing.Revision(ref))
	if err != nil {
		hsh, err = r.ResolveRevision(plumbing.Revision(ref))
		if err != nil {
			return nil, err
		}
	}

	return hsh, nil
}
