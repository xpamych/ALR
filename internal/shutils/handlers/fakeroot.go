// This file was originally part of the project "LURE - Linux User REpository", created by Elara Musayelyan.
// It has been modified as part of "ALR - Any Linux Repository" by Евгений Храмов.
//
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

package handlers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

// FakerootExecHandler was extracted from github.com/mvdan/sh/interp/handler.go
// and modified to run commands in a fakeroot environent.
func FakerootExecHandler(killTimeout time.Duration) interp.ExecHandlerFunc {
	return func(ctx context.Context, args []string) error {
		hc := interp.HandlerCtx(ctx)
		path, err := interp.LookPathDir(hc.Dir, hc.Env, args[0])
		if err != nil {
			fmt.Fprintln(hc.Stderr, err)
			return interp.NewExitStatus(127)
		}
		cmd := &exec.Cmd{
			Path:   path,
			Args:   args,
			Env:    execEnv(hc.Env),
			Dir:    hc.Dir,
			Stdin:  hc.Stdin,
			Stdout: hc.Stdout,
			Stderr: hc.Stderr,
		}

		err = Apply(cmd)
		if err != nil {
			return err
		}

		err = cmd.Start()
		if err == nil {
			if done := ctx.Done(); done != nil {
				go func() {
					<-done

					if killTimeout <= 0 || runtime.GOOS == "windows" {
						_ = cmd.Process.Signal(os.Kill)
						return
					}

					// TODO: don't temporarily leak this goroutine
					// if the program stops itself with the
					// interrupt.
					go func() {
						time.Sleep(killTimeout)
						_ = cmd.Process.Signal(os.Kill)
					}()
					_ = cmd.Process.Signal(os.Interrupt)
				}()
			}

			err = cmd.Wait()
		}

		switch x := err.(type) {
		case *exec.ExitError:
			// started, but errored - default to 1 if OS
			// doesn't have exit statuses
			if status, ok := x.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					return interp.NewExitStatus(uint8(128 + status.Signal()))
				}
				return interp.NewExitStatus(uint8(status.ExitStatus()))
			}
			return interp.NewExitStatus(1)
		case *exec.Error:
			// did not start
			fmt.Fprintf(hc.Stderr, "%v\n", err)
			return interp.NewExitStatus(127)
		default:
			return err
		}
	}
}

func rootMap(m syscall.SysProcIDMap) bool {
	return m.ContainerID == 0
}

func Apply(cmd *exec.Cmd) error {
	uid := os.Getuid()
	gid := os.Getgid()

	// If the user is already root, there's no need for fakeroot
	if uid == 0 {
		return nil
	}

	// Ensure SysProcAttr isn't nil
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// Create a new user namespace
	cmd.SysProcAttr.Cloneflags |= syscall.CLONE_NEWUSER

	// If the command already contains a mapping for the root user, return an error
	if slices.ContainsFunc(cmd.SysProcAttr.UidMappings, rootMap) {
		return nil
	}

	// If the command already contains a mapping for the root group, return an error
	if slices.ContainsFunc(cmd.SysProcAttr.GidMappings, rootMap) {
		return nil
	}

	cmd.SysProcAttr.UidMappings = append(cmd.SysProcAttr.UidMappings, syscall.SysProcIDMap{
		ContainerID: 0,
		HostID:      uid,
		Size:        1,
	})

	cmd.SysProcAttr.GidMappings = append(cmd.SysProcAttr.GidMappings, syscall.SysProcIDMap{
		ContainerID: 0,
		HostID:      gid,
		Size:        1,
	})

	return nil
}

// execEnv was extracted from github.com/mvdan/sh/interp/vars.go
func execEnv(env expand.Environ) []string {
	list := make([]string, 0, 64)
	env.Each(func(name string, vr expand.Variable) bool {
		if !vr.IsSet() {
			// If a variable is set globally but unset in the
			// runner, we need to ensure it's not part of the final
			// list. Seems like zeroing the element is enough.
			// This is a linear search, but this scenario should be
			// rare, and the number of variables shouldn't be large.
			for i, kv := range list {
				if strings.HasPrefix(kv, name+"=") {
					list[i] = ""
				}
			}
		}
		if vr.Exported && vr.Kind == expand.String {
			list = append(list, name+"="+vr.String())
		}
		return true
	})
	return list
}
