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

package build

import (
	"fmt"
	"log/slog"
	"net/rpc"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/hashicorp/go-plugin"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/logger"
)

type InstallerPlugin struct {
	Impl InstallerExecutor
}

type InstallerRPC struct {
	client *rpc.Client
}

type InstallerRPCServer struct {
	Impl InstallerExecutor
}

func (r *InstallerRPC) InstallLocal(paths []string) error {
	return r.client.Call("Plugin.InstallLocal", paths, nil)
}

func (s *InstallerRPCServer) InstallLocal(paths []string, reply *struct{}) error {
	return s.Impl.InstallLocal(paths)
}

func (r *InstallerRPC) Install(pkgs []string) error {
	return r.client.Call("Plugin.Install", pkgs, nil)
}

func (s *InstallerRPCServer) Install(pkgs []string, reply *struct{}) error {
	slog.Debug("install", "pkgs", pkgs)
	return s.Impl.Install(pkgs)
}

func (r *InstallerRPC) RemoveAlreadyInstalled(paths []string) ([]string, error) {
	var val []string
	err := r.client.Call("Plugin.RemoveAlreadyInstalled", paths, &val)
	return val, err
}

func (s *InstallerRPCServer) RemoveAlreadyInstalled(pkgs []string, res *[]string) error {
	vars, err := s.Impl.RemoveAlreadyInstalled(pkgs)
	if err != nil {
		return err
	}
	*res = vars
	return nil
}

func (p *InstallerPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &InstallerRPC{client: c}, nil
}

func (p *InstallerPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &InstallerRPCServer{Impl: p.Impl}, nil
}

func GetSafeInstaller() (InstallerExecutor, func(), error) {
	var err error

	executable, err := os.Executable()
	if err != nil {
		return nil, nil, err
	}
	cmd := exec.Command(executable, "_internal-installer")
	setCommonCmdEnv(cmd)

	slog.Debug("safe installer setup", "uid", syscall.Getuid(), "gid", syscall.Getgid())

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         pluginMap,
		Cmd:             cmd,
		Logger:          logger.GetHCLoggerAdapter(),
		SkipHostEnv:     true,
		UnixSocketConfig: &plugin.UnixSocketConfig{
			Group: "alr",
		},
		SyncStderr: os.Stderr,
	})
	rpcClient, err := client.Client()
	if err != nil {
		return nil, nil, err
	}

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			client.Kill()
		})
	}

	defer func() {
		if err != nil {
			slog.Debug("close installer")
			cleanup()
		}
	}()

	raw, err := rpcClient.Dispense("installer")
	if err != nil {
		return nil, nil, err
	}

	executor, ok := raw.(InstallerExecutor)
	if !ok {
		err = fmt.Errorf("dispensed object is not a ScriptExecutor (got %T)", raw)
		return nil, nil, err
	}

	return executor, cleanup, nil
}
