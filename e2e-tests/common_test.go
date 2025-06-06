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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/efficientgo/e2e"
	"github.com/stretchr/testify/assert"

	expect "github.com/tailscale/goexpect"
)

// DebugWriter оборачивает io.Writer и логирует все записываемые данные.
type DebugWriter struct {
	prefix string
	writer io.Writer
}

func (d *DebugWriter) Write(p []byte) (n int, err error) {
	log.Printf("%s: Writing data: %q", d.prefix, p) // Логируем данные
	return d.writer.Write(p)
}

// DebugReader оборачивает io.Reader и логирует все читаемые данные.
type DebugReader struct {
	prefix string
	reader io.Reader
}

func (d *DebugReader) Read(p []byte) (n int, err error) {
	n, err = d.reader.Read(p)
	if n > 0 {
		log.Printf("%s: Read data: %q", d.prefix, p[:n]) // Логируем данные
	}
	return n, err
}

func e2eSpawn(runnable e2e.Runnable, command e2e.Command, timeout time.Duration, opts ...expect.Option) (expect.Expecter, <-chan error, error, *io.PipeWriter) {
	resCh := make(chan error)

	// Создаем pipe для stdin и stdout
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()

	debugStdinReader := &DebugReader{prefix: "STDIN", reader: stdinReader}
	debugStdoutWriter := &DebugWriter{prefix: "STDOUT", writer: stdoutWriter}

	go func() {
		err := runnable.Exec(
			command,
			e2e.WithExecOptionStdout(debugStdoutWriter),
			e2e.WithExecOptionStdin(debugStdinReader),
			e2e.WithExecOptionStderr(debugStdoutWriter),
		)

		resCh <- err
	}()

	exp, chnErr, err := expect.SpawnGeneric(&expect.GenOptions{
		In:  stdinWriter,
		Out: stdoutReader,
		Wait: func() error {
			return <-resCh
		},
		Close: func() error {
			stdinWriter.Close()
			stdoutReader.Close()
			return nil
		},
		Check: func() bool { return true },
	}, timeout, expect.Verbose(true), expect.VerboseWriter(os.Stdout))

	return exp, chnErr, err, stdinWriter
}

var ALL_SYSTEMS []string = []string{
	"ubuntu-24.04",
	"alt-sisyphus",
	"fedora-41",
	// "archlinux",
	// "alpine",
	// "opensuse-leap",
	// "redos-8",
}

var AUTOREQ_AUTOPROV_SYSTEMS []string = []string{
	// "alt-sisyphus",
	"fedora-41",
}

var RPM_SYSTEMS []string = []string{
	"fedora-41",
}

var COMMON_SYSTEMS []string = []string{
	"ubuntu-24.04",
}

func dockerMultipleRun(t *testing.T, name string, ids []string, f func(t *testing.T, runnable e2e.Runnable)) {
	t.Run(name, func(t *testing.T) {
		for _, id := range ids {
			t.Run(id, func(t *testing.T) {
				t.Parallel()
				dockerName := fmt.Sprintf("alr-test-%s-%s", name, id)
				hash := sha256.New()
				hash.Write([]byte(dockerName))
				hashSum := hash.Sum(nil)
				hashString := hex.EncodeToString(hashSum)
				truncatedHash := hashString[:8]
				e, err := e2e.New(e2e.WithVerbose(), e2e.WithName(fmt.Sprintf("alr-%s", truncatedHash)))
				assert.NoError(t, err)
				t.Cleanup(e.Close)
				imageId := fmt.Sprintf("ghcr.io/maks1ms/alr-e2e-test-image-%s", id)
				runnable := e.Runnable(dockerName).Init(
					e2e.StartOptions{
						Image: imageId,
						Volumes: []string{
							"./alr:/tmp/alr",
						},
						Privileged: true,
					},
				)
				assert.NoError(t, e2e.StartAndWaitReady(runnable))
				err = runnable.Exec(e2e.NewCommand("/bin/alr-test-setup", "alr-install"))
				if err != nil {
					panic(err)
				}
				err = runnable.Exec(e2e.NewCommand("/bin/alr-test-setup", "passwordless-sudo-setup"))
				if err != nil {
					panic(err)
				}
				f(t, runnable)
			})
		}
	})
}

func execShouldNoError(t *testing.T, r e2e.Runnable, cmd string, args ...string) {
	assert.NoError(t, r.Exec(e2e.NewCommand(cmd, args...)))
}

func execShouldError(t *testing.T, r e2e.Runnable, cmd string, args ...string) {
	assert.Error(t, r.Exec(e2e.NewCommand(cmd, args...)))
}

func runTestCommands(t *testing.T, r e2e.Runnable, timeout time.Duration, expects []expect.Batcher) {
	exp, _, err, _ := e2eSpawn(
		r,
		e2e.NewCommand("/bin/bash"), 25*time.Second,
		expect.Verbose(true),
	)
	assert.NoError(t, err)
	_, err = exp.ExpectBatch(
		expects,
		timeout,
	)
	assert.NoError(t, err)
}

const REPO_NAME_FOR_E2E_TESTS = "alr-repo"
const REPO_URL_FOR_E2E_TESTS = "https://gitea.plemya-x.ru/Plemya-x/repo-for-tests.git"

func defaultPrepare(t *testing.T, r e2e.Runnable) {
	execShouldNoError(t, r,
		"sudo",
		"alr",
		"repo",
		"add",
		REPO_NAME_FOR_E2E_TESTS,
		REPO_URL_FOR_E2E_TESTS,
	)

	execShouldNoError(t, r,
		"sudo",
		"alr",
		"ref",
	)
}
