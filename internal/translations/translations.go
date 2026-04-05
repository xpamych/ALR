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

package translations

import (
	"embed"

	"github.com/jeandeaual/go-locale"
	"github.com/leonelquinteros/gotext"
)

//go:embed po
var poFS embed.FS

// Supported languages
const (
	LangRussian = "ru"
	LangEnglish = "en"
	LangDefault = LangRussian // Russian is default
)

func Setup() {
	userLanguage, err := locale.GetLanguage()
	if err != nil {
		userLanguage = LangDefault
	}

	// Check if user language is supported
	supportedLangs := map[string]bool{
		LangRussian: true,
		LangEnglish: true,
	}

	// If user language is not supported, use default (Russian)
	if !supportedLangs[userLanguage] {
		userLanguage = LangDefault
	}

	// Create locales for all supported languages
	var locales []*gotext.Locale

	// Add user language first (primary)
	loc := gotext.NewLocaleFSWithPath(userLanguage, &poFS, "po")
	loc.SetDomain("default")
	locales = append(locales, loc)

	// Add fallback locale (Russian) if primary is different
	if userLanguage != LangDefault {
		fallbackLoc := gotext.NewLocaleFSWithPath(LangDefault, &poFS, "po")
		fallbackLoc.SetDomain("default")
		locales = append(locales, fallbackLoc)
	}

	gotext.SetLocales(locales)
}
