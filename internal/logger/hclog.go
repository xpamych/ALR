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

package logger

import (
	"io"
	"log"
	"strings"

	chLog "github.com/charmbracelet/log"
	"github.com/hashicorp/go-hclog"
)

type HCLoggerAdapter struct {
	logger *Logger
}

func hclogLevelTochLog(level hclog.Level) chLog.Level {
	switch level {
	case hclog.Debug:
		return chLog.DebugLevel
	case hclog.Info:
		return chLog.InfoLevel
	case hclog.Warn:
		return chLog.WarnLevel
	case hclog.Error:
		return chLog.ErrorLevel
	}
	return chLog.FatalLevel
}

func (a *HCLoggerAdapter) Log(level hclog.Level, msg string, args ...interface{}) {
	filteredArgs := make([]interface{}, 0, len(args))
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			filteredArgs = append(filteredArgs, args[i])
			continue
		}

		key, ok := args[i].(string)
		if !ok || key != "timestamp" {
			filteredArgs = append(filteredArgs, args[i], args[i+1])
		}
	}

	// Start ugly hacks
	// Ignore exit messages
	// - https://github.com/hashicorp/go-plugin/issues/331
	// - https://github.com/hashicorp/go-plugin/issues/203
	// - https://github.com/hashicorp/go-plugin/issues/192
	var chLogLevel chLog.Level
	if msg == "plugin process exited" ||
		strings.HasPrefix(msg, "[ERR] plugin: stream copy 'stderr' error") ||
		strings.HasPrefix(msg, "[DEBUG] plugin") {
		chLogLevel = chLog.DebugLevel
	} else {
		chLogLevel = hclogLevelTochLog(level)
	}

	a.logger.l.Log(chLogLevel, msg, filteredArgs...)
}

func (a *HCLoggerAdapter) Trace(msg string, args ...interface{}) {
	a.Log(hclog.Trace, msg, args...)
}

func (a *HCLoggerAdapter) Debug(msg string, args ...interface{}) {
	a.Log(hclog.Debug, msg, args...)
}

func (a *HCLoggerAdapter) Info(msg string, args ...interface{}) {
	a.Log(hclog.Info, msg, args...)
}

func (a *HCLoggerAdapter) Warn(msg string, args ...interface{}) {
	a.Log(hclog.Warn, msg, args...)
}

func (a *HCLoggerAdapter) Error(msg string, args ...interface{}) {
	a.Log(hclog.Error, msg, args...)
}

func (a *HCLoggerAdapter) IsTrace() bool {
	return a.logger.l.GetLevel() <= chLog.DebugLevel
}

func (a *HCLoggerAdapter) IsDebug() bool {
	return a.logger.l.GetLevel() <= chLog.DebugLevel
}

func (a *HCLoggerAdapter) IsInfo() bool {
	return a.logger.l.GetLevel() <= chLog.InfoLevel
}

func (a *HCLoggerAdapter) IsWarn() bool {
	return a.logger.l.GetLevel() <= chLog.WarnLevel
}

func (a *HCLoggerAdapter) IsError() bool {
	return a.logger.l.GetLevel() <= chLog.ErrorLevel
}

func (a *HCLoggerAdapter) ImpliedArgs() []interface{} {
	return nil
}

func (a *HCLoggerAdapter) With(args ...interface{}) hclog.Logger {
	return a
}

func (a *HCLoggerAdapter) Name() string {
	return ""
}

func (a *HCLoggerAdapter) Named(name string) hclog.Logger {
	return a
}

func (a *HCLoggerAdapter) ResetNamed(name string) hclog.Logger {
	return a
}

func (a *HCLoggerAdapter) SetLevel(level hclog.Level) {
}

func (a *HCLoggerAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return nil
}

func (a *HCLoggerAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return nil
}

func GetHCLoggerAdapter() *HCLoggerAdapter {
	return &HCLoggerAdapter{
		logger: logger,
	}
}
