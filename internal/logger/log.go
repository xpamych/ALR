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

package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/leonelquinteros/gotext"
)

type Logger struct {
	lOut slog.Handler
	lErr slog.Handler
}

func setupOutLogger() *log.Logger {
	styles := log.DefaultStyles()
	logger := log.New(os.Stdout)
	logger.SetStyles(styles)
	return logger
}

func setupErrorLogger() *log.Logger {
	styles := log.DefaultStyles()
	styles.Levels[log.ErrorLevel] = lipgloss.NewStyle().
		SetString(gotext.Get("ERROR")).
		Padding(0, 1, 0, 1).
		Background(lipgloss.Color("204")).
		Foreground(lipgloss.Color("0"))
	logger := log.New(os.Stderr)
	logger.SetStyles(styles)
	return logger
}

func New() *Logger {
	standardLogger := setupOutLogger()
	errLogger := setupErrorLogger()
	return &Logger{
		lOut: standardLogger,
		lErr: errLogger,
	}
}

func (l *Logger) Enabled(ctx context.Context, level slog.Level) bool {
	if level <= slog.LevelInfo {
		return l.lOut.Enabled(ctx, level)
	}
	return l.lErr.Enabled(ctx, level)
}

func (l *Logger) Handle(ctx context.Context, rec slog.Record) error {
	if rec.Level <= slog.LevelInfo {
		return l.lOut.Handle(ctx, rec)
	}
	return l.lErr.Handle(ctx, rec)
}

func (l *Logger) WithAttrs(attrs []slog.Attr) slog.Handler {
	sl := *l
	sl.lOut = l.lOut.WithAttrs(attrs)
	sl.lErr = l.lErr.WithAttrs(attrs)
	return &sl
}

func (l *Logger) WithGroup(name string) slog.Handler {
	sl := *l
	sl.lOut = l.lOut.WithGroup(name)
	sl.lErr = l.lErr.WithGroup(name)
	return &sl
}

func SetupDefault() {
	logger := slog.New(New())
	slog.SetDefault(logger)
}
