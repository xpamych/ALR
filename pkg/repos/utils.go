package repos

import (
	"context"
	"io"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
	"plemya-x.ru/alr/internal/db"
	"plemya-x.ru/alr/internal/shutils/decoder"
	"plemya-x.ru/alr/pkg/distro"
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

var overridable = map[string]string{
	"deps":       "Depends",
	"build_deps": "BuildDepends",
	"desc":       "Description",
	"homepage":   "Homepage",
	"maintainer": "Maintainer",
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
