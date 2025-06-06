// This file was originally part of the project "LURE - Linux User REpository", created by Elara Musayelyan.
// It has been modified as part of "ALR - Any Linux Repository" by the ALR Authors.
//
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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	gitConfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/leonelquinteros/gotext"
	"github.com/pelletier/go-toml/v2"
	"go.elara.ws/vercmp"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/config"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/shutils/handlers"
	"gitea.plemya-x.ru/Plemya-x/ALR/internal/types"
)

type actionType uint8

const (
	actionDelete actionType = iota
	actionUpdate
)

type action struct {
	Type actionType
	File string
}

// Pull pulls the provided repositories. If a repo doesn't exist, it will be cloned
// and its packages will be written to the DB. If it does exist, it will be pulled.
// In this case, only changed packages will be processed if possible.
// If repos is set to nil, the repos in the ALR config will be used.
func (rs *Repos) Pull(ctx context.Context, repos []types.Repo) error {
	if repos == nil {
		repos = rs.cfg.Repos()
	}

	for _, repo := range repos {
		repoURL, err := url.Parse(repo.URL)
		if err != nil {
			return err
		}

		slog.Info(gotext.Get("Pulling repository"), "name", repo.Name)
		repoDir := filepath.Join(rs.cfg.GetPaths().RepoDir, repo.Name)

		var repoFS billy.Filesystem
		gitDir := filepath.Join(repoDir, ".git")
		// Only pull repos that contain valid git repos
		if fi, err := os.Stat(gitDir); err == nil && fi.IsDir() {
			r, err := git.PlainOpen(repoDir)
			if err != nil {
				return err
			}

			err = r.FetchContext(ctx, &git.FetchOptions{
				Progress: os.Stderr,
				Force:    true,
			})
			if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
				return err
			}

			w, err := r.Worktree()
			if err != nil {
				return err
			}

			old, err := r.Head()
			if err != nil {
				return err
			}

			revHash, err := resolveHash(r, repo.Ref)
			if err != nil {
				return fmt.Errorf("error resolving hash: %w", err)
			}

			if old.Hash() == *revHash {
				slog.Info(gotext.Get("Repository up to date"), "name", repo.Name)
			}

			err = w.Checkout(&git.CheckoutOptions{
				Hash:  plumbing.NewHash(revHash.String()),
				Force: true,
			})
			if err != nil {
				return err
			}
			repoFS = w.Filesystem

			new, err := r.Head()
			if err != nil {
				return err
			}

			// If the DB was not present at startup, that means it's
			// empty. In this case, we need to update the DB fully
			// rather than just incrementally.
			if rs.db.IsEmpty(ctx) {
				err = rs.processRepoFull(ctx, repo, repoDir)
				if err != nil {
					return err
				}
			} else {
				err = rs.processRepoChanges(ctx, repo, r, w, old, new)
				if err != nil {
					return err
				}
			}
		} else {
			err = os.RemoveAll(repoDir)
			if err != nil {
				return err
			}

			err = os.MkdirAll(repoDir, 0o755)
			if err != nil {
				return err
			}

			r, err := git.PlainInit(repoDir, false)
			if err != nil {
				return err
			}

			_, err = r.CreateRemote(&gitConfig.RemoteConfig{
				Name: git.DefaultRemoteName,
				URLs: []string{repoURL.String()},
			})
			if err != nil {
				return err
			}

			err = r.FetchContext(ctx, &git.FetchOptions{
				Progress: os.Stderr,
				Force:    true,
			})
			if err != nil {
				return err
			}

			w, err := r.Worktree()
			if err != nil {
				return err
			}

			revHash, err := resolveHash(r, repo.Ref)
			if err != nil {
				return fmt.Errorf("error resolving hash: %w", err)
			}

			err = w.Checkout(&git.CheckoutOptions{
				Hash:  plumbing.NewHash(revHash.String()),
				Force: true,
			})
			if err != nil {
				return err
			}

			err = rs.processRepoFull(ctx, repo, repoDir)
			if err != nil {
				return err
			}

			repoFS = osfs.New(repoDir)
		}

		fl, err := repoFS.Open("alr-repo.toml")
		if err != nil {
			slog.Warn(gotext.Get("Git repository does not appear to be a valid ALR repo"), "repo", repo.Name)
			continue
		}

		var repoCfg types.RepoConfig
		err = toml.NewDecoder(fl).Decode(&repoCfg)
		if err != nil {
			return err
		}
		fl.Close()

		// If the version doesn't have a "v" prefix, it's not a standard version.
		// It may be "unknown" or a git version, but either way, there's no way
		// to compare it to the repo version, so only compare versions with the "v".
		if strings.HasPrefix(config.Version, "v") {
			if vercmp.Compare(config.Version, repoCfg.Repo.MinVersion) == -1 {
				slog.Warn(gotext.Get("ALR repo's minimum ALR version is greater than the current version. Try updating ALR if something doesn't work."), "repo", repo.Name)
			}
		}
	}

	return nil
}

func (rs *Repos) updatePkg(ctx context.Context, repo types.Repo, runner *interp.Runner, scriptFl io.ReadCloser) error {
	parser := syntax.NewParser()

	pkgs, err := parseScript(ctx, repo, parser, runner, scriptFl)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		err = rs.db.InsertPackage(ctx, *pkg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rs *Repos) processRepoChangesRunner(repoDir, scriptDir string) (*interp.Runner, error) {
	env := append(os.Environ(), "scriptdir="+scriptDir)
	return interp.New(
		interp.Env(expand.ListEnviron(env...)),
		interp.ExecHandler(handlers.NopExec),
		interp.ReadDirHandler2(handlers.RestrictedReadDir(repoDir)),
		interp.StatHandler(handlers.RestrictedStat(repoDir)),
		interp.OpenHandler(handlers.RestrictedOpen(repoDir)),
		interp.StdIO(handlers.NopRWC{}, handlers.NopRWC{}, handlers.NopRWC{}),
		// Use temp dir instead script dir because runner may be for deleted file
		interp.Dir(os.TempDir()),
	)
}

func (rs *Repos) processRepoChanges(ctx context.Context, repo types.Repo, r *git.Repository, w *git.Worktree, old, new *plumbing.Reference) error {
	oldCommit, err := r.CommitObject(old.Hash())
	if err != nil {
		return err
	}

	newCommit, err := r.CommitObject(new.Hash())
	if err != nil {
		return err
	}

	patch, err := oldCommit.Patch(newCommit)
	if err != nil {
		return fmt.Errorf("error to create patch: %w", err)
	}

	var actions []action
	for _, fp := range patch.FilePatches() {
		from, to := fp.Files()

		if !isValid(from, to) {
			continue
		}

		switch {
		case to == nil:
			actions = append(actions, action{
				Type: actionDelete,
				File: from.Path(),
			})
		case from == nil:
			actions = append(actions, action{
				Type: actionUpdate,
				File: to.Path(),
			})
		case from.Path() != to.Path():
			actions = append(actions,
				action{
					Type: actionDelete,
					File: from.Path(),
				},
				action{
					Type: actionUpdate,
					File: to.Path(),
				},
			)
		default:
			slog.Debug("unexpected, but I'll try to do")
			actions = append(actions, action{
				Type: actionUpdate,
				File: to.Path(),
			})
		}
	}

	repoDir := w.Filesystem.Root()
	parser := syntax.NewParser()

	for _, action := range actions {
		runner, err := rs.processRepoChangesRunner(repoDir, filepath.Dir(filepath.Join(repoDir, action.File)))
		if err != nil {
			return fmt.Errorf("error creating process repo changes runner: %w", err)
		}

		switch action.Type {
		case actionDelete:
			if filepath.Base(action.File) != "alr.sh" {
				continue
			}
			scriptFl, err := oldCommit.File(action.File)
			if err != nil {
				return nil
			}

			r, err := scriptFl.Reader()
			if err != nil {
				return nil
			}

			pkgs, err := parseScript(ctx, repo, parser, runner, r)
			if err != nil {
				return err
			}

			for _, pkg := range pkgs {
				err = rs.db.DeletePkgs(ctx, "name = ? AND repository = ?", pkg.Name, repo.Name)
				if err != nil {
					return err
				}
			}
		case actionUpdate:
			if filepath.Base(action.File) != "alr.sh" {
				action.File = filepath.Join(filepath.Dir(action.File), "alr.sh")
			}

			scriptFl, err := newCommit.File(action.File)
			if err != nil {
				return nil
			}

			r, err := scriptFl.Reader()
			if err != nil {
				return nil
			}

			err = rs.updatePkg(ctx, repo, runner, r)
			if err != nil {
				return fmt.Errorf("error updatePkg: %w", err)
			}
		}
	}

	return nil
}

func (rs *Repos) processRepoFull(ctx context.Context, repo types.Repo, repoDir string) error {
	glob := filepath.Join(repoDir, "/*/alr.sh")
	matches, err := filepath.Glob(glob)
	if err != nil {
		return err
	}

	for _, match := range matches {
		runner, err := rs.processRepoChangesRunner(repoDir, filepath.Dir(match))
		if err != nil {
			return err
		}

		scriptFl, err := os.Open(match)
		if err != nil {
			return err
		}

		err = rs.updatePkg(ctx, repo, runner, scriptFl)
		if err != nil {
			return err
		}
	}

	return nil
}
