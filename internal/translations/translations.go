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

package translations

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"sync"

	"github.com/jeandeaual/go-locale"
	"github.com/leonelquinteros/gotext"
	"go.elara.ws/logger"
	"go.elara.ws/translate"
	"golang.org/x/text/language"
)

//go:embed files
var translationFS embed.FS

var (
	mu         sync.Mutex
	translator *translate.Translator
)

func Translator(ctx context.Context) *translate.Translator {
	mu.Lock()
	defer mu.Unlock()
	if translator == nil {
		t, err := translate.NewFromFS(translationFS)
		if err != nil {
			slog.Error(gotext.Get("Error creating new translator"), "err", err)
			os.Exit(1)
		}
		translator = &t
	}
	return translator
}

func NewLogger(ctx context.Context, l logger.Logger, lang language.Tag) *translate.TranslatedLogger {
	return translate.NewLogger(l, *Translator(ctx), lang)
}

//go:embed po
var poFS embed.FS

func Setup() {
	userLanguage, err := locale.GetLanguage()
	if err != nil {
		panic(err)
	}

	_, err = fs.Stat(poFS, path.Join("po", userLanguage))
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		panic(err)
	}

	loc := gotext.NewLocaleFSWithPath(userLanguage, &poFS, "po")
	loc.SetDomain("default")
	gotext.SetLocales([]*gotext.Locale{loc})
}
