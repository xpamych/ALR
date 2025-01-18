package build

import (
	"bytes"
	"context"
	"os/exec"
	"path"
	"strings"

	"github.com/goreleaser/nfpm/v2"
	"plemya-x.ru/alr/internal/types"
	"plemya-x.ru/alr/pkg/loggerctx"
)

func rpmFindDependencies(ctx context.Context, pkgInfo *nfpm.Info, dirs types.Directories, command string, updateFunc func(string)) error {
	log := loggerctx.From(ctx)

	if _, err := exec.LookPath(command); err != nil {
		log.Info("Command not found on the system").Str("command", command).Send()
		return nil
	}

	var paths []string
	for _, content := range pkgInfo.Contents {
		if content.Type != "dir" {
			paths = append(paths,
				path.Join(dirs.PkgDir, content.Destination),
			)
		}
	}

	if len(paths) == 0 {
		return nil
	}

	cmd := exec.Command(command)
	cmd.Stdin = bytes.NewBufferString(strings.Join(paths, "\n"))
	cmd.Env = append(cmd.Env,
		"RPM_BUILD_ROOT="+dirs.PkgDir,
		"RPM_FINDPROV_METHOD=",
		"RPM_FINDREQ_METHOD=",
		"RPM_DATADIR=",
		"RPM_SUBPACKAGE_NAME=",
	)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Error(stderr.String()).Send()
		return err
	}

	dependencies := strings.Split(strings.TrimSpace(out.String()), "\n")
	for _, dep := range dependencies {
		if dep != "" {
			updateFunc(dep)
		}
	}

	return nil
}

func rpmFindProvides(ctx context.Context, pkgInfo *nfpm.Info, dirs types.Directories) error {
	log := loggerctx.From(ctx)

	return rpmFindDependencies(ctx, pkgInfo, dirs, "/usr/lib/rpm/find-provides", func(dep string) {
		log.Info("Provided dependency found").Str("dep", dep).Send()
		pkgInfo.Overridables.Provides = append(pkgInfo.Overridables.Provides, dep)
	})
}

func rpmFindRequires(ctx context.Context, pkgInfo *nfpm.Info, dirs types.Directories) error {
	log := loggerctx.From(ctx)

	return rpmFindDependencies(ctx, pkgInfo, dirs, "/usr/lib/rpm/find-requires", func(dep string) {
		log.Info("Required dependency found").Str("dep", dep).Send()
		pkgInfo.Overridables.Depends = append(pkgInfo.Overridables.Depends, dep)
	})
}
