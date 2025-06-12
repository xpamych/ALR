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
	"fmt"
	"log/slog"
	"net/rpc"
	"os"
	"os/exec"
	"sync"

	"github.com/hashicorp/go-plugin"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/logger"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/alrsh"
)

var HandshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "ALR_PLUGIN",
	MagicCookieValue: "-",
}

type ScriptExecutorPlugin struct {
	Impl ScriptExecutor
}

type ScriptExecutorRPCServer struct {
	Impl ScriptExecutor
}

// =============================
//
// ReadScript
//

func (s *ScriptExecutorRPC) ReadScript(ctx context.Context, scriptPath string) (*alrsh.ScriptFile, error) {
	var resp *alrsh.ScriptFile
	err := s.client.Call("Plugin.ReadScript", scriptPath, &resp)
	return resp, err
}

func (s *ScriptExecutorRPCServer) ReadScript(scriptPath string, resp *alrsh.ScriptFile) error {
	file, err := s.Impl.ReadScript(context.Background(), scriptPath)
	if err != nil {
		return err
	}
	*resp = *file
	return nil
}

// =============================
//
// ExecuteFirstPass
//

type ExecuteFirstPassArgs struct {
	Input *BuildInput
	Sf    *alrsh.ScriptFile
}

type ExecuteFirstPassResp struct {
	BasePkg        string
	VarsOfPackages []*alrsh.Package
}

func (s *ScriptExecutorRPC) ExecuteFirstPass(ctx context.Context, input *BuildInput, sf *alrsh.ScriptFile) (string, []*alrsh.Package, error) {
	var resp *ExecuteFirstPassResp
	err := s.client.Call("Plugin.ExecuteFirstPass", &ExecuteFirstPassArgs{
		Input: input,
		Sf:    sf,
	}, &resp)
	if err != nil {
		return "", nil, err
	}
	return resp.BasePkg, resp.VarsOfPackages, nil
}

func (s *ScriptExecutorRPCServer) ExecuteFirstPass(args *ExecuteFirstPassArgs, resp *ExecuteFirstPassResp) error {
	basePkg, varsOfPackages, err := s.Impl.ExecuteFirstPass(context.Background(), args.Input, args.Sf)
	if err != nil {
		return err
	}
	*resp = ExecuteFirstPassResp{
		BasePkg:        basePkg,
		VarsOfPackages: varsOfPackages,
	}
	return nil
}

// =============================
//
// PrepareDirs
//

type PrepareDirsArgs struct {
	Input   *BuildInput
	BasePkg string
}

func (s *ScriptExecutorRPC) PrepareDirs(
	ctx context.Context,
	input *BuildInput,
	basePkg string,
) error {
	err := s.client.Call("Plugin.PrepareDirs", &PrepareDirsArgs{
		Input:   input,
		BasePkg: basePkg,
	}, nil)
	if err != nil {
		return err
	}
	return err
}

func (s *ScriptExecutorRPCServer) PrepareDirs(args *PrepareDirsArgs, reply *struct{}) error {
	err := s.Impl.PrepareDirs(
		context.Background(),
		args.Input,
		args.BasePkg,
	)
	if err != nil {
		return err
	}
	return err
}

// =============================
//
// ExecuteSecondPass
//

type ExecuteSecondPassArgs struct {
	Input          *BuildInput
	Sf             *alrsh.ScriptFile
	VarsOfPackages []*alrsh.Package
	RepoDeps       []string
	BuiltDeps      []*BuiltDep
	BasePkg        string
}

func (s *ScriptExecutorRPC) ExecuteSecondPass(
	ctx context.Context,
	input *BuildInput,
	sf *alrsh.ScriptFile,
	varsOfPackages []*alrsh.Package,
	repoDeps []string,
	builtDeps []*BuiltDep,
	basePkg string,
) ([]*BuiltDep, error) {
	var resp []*BuiltDep
	err := s.client.Call("Plugin.ExecuteSecondPass", &ExecuteSecondPassArgs{
		Input:          input,
		Sf:             sf,
		VarsOfPackages: varsOfPackages,
		RepoDeps:       repoDeps,
		BuiltDeps:      builtDeps,
		BasePkg:        basePkg,
	}, &resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *ScriptExecutorRPCServer) ExecuteSecondPass(args *ExecuteSecondPassArgs, resp *[]*BuiltDep) error {
	res, err := s.Impl.ExecuteSecondPass(
		context.Background(),
		args.Input,
		args.Sf,
		args.VarsOfPackages,
		args.RepoDeps,
		args.BuiltDeps,
		args.BasePkg,
	)
	if err != nil {
		return err
	}
	*resp = res
	return err
}

//
// ============================
//

func (p *ScriptExecutorPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &ScriptExecutorRPCServer{Impl: p.Impl}, nil
}

func (p *ScriptExecutorPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &ScriptExecutorRPC{client: c}, nil
}

type ScriptExecutorRPC struct {
	client *rpc.Client
}

var pluginMap = map[string]plugin.Plugin{
	"script-executor": &ScriptExecutorPlugin{},
	"installer":       &InstallerPlugin{},
}

func GetSafeScriptExecutor() (ScriptExecutor, func(), error) {
	var err error

	executable, err := os.Executable()
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.Command(executable, "_internal-safe-script-executor")
	setCommonCmdEnv(cmd)

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
			slog.Debug("close script-executor")
			cleanup()
		}
	}()

	raw, err := rpcClient.Dispense("script-executor")
	if err != nil {
		return nil, nil, err
	}

	executor, ok := raw.(ScriptExecutor)
	if !ok {
		err = fmt.Errorf("dispensed object is not a ScriptExecutor (got %T)", raw)
		return nil, nil, err
	}

	return executor, cleanup, nil
}
