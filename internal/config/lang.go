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

package config

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/leonelquinteros/gotext"
	"golang.org/x/text/language"
)

var (
	langMtx sync.Mutex
	lang    language.Tag
	langSet bool
)

// Language returns the system language.
// The first time it's called, it'll detect the langauge based on
// the $LANG environment variable.
// Subsequent calls will just return the same value.
func Language(ctx context.Context) language.Tag {
	langMtx.Lock()
	defer langMtx.Unlock()
	if !langSet {
		syslang := SystemLang()
		tag, err := language.Parse(syslang)
		if err != nil {
			slog.Error(gotext.Get("Error parsing system language"), "err", err)
			os.Exit(1)
		}
		base, _ := tag.Base()
		lang = language.Make(base.String())
		langSet = true
	}
	return lang
}

// SystemLang returns the system language based on
// the $LANG environment variable.
func SystemLang() string {
	lang := os.Getenv("LANG")
	lang, _, _ = strings.Cut(lang, ".")
	if lang == "" || lang == "C" {
		lang = "en"
	}
	return lang
}
