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

package cliutils

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/leonelquinteros/gotext"

	"gitea.plemya-x.ru/Plemya-x/ALR/internal/pager"
	"gitea.plemya-x.ru/Plemya-x/ALR/pkg/alrsh"
)

// YesNoPrompt asks the user a yes or no question, using def as the default answer
func YesNoPrompt(ctx context.Context, msg string, interactive, def bool) (bool, error) {
	if interactive {
		var answer bool
		err := survey.AskOne(
			&survey.Confirm{
				Message: msg,
				Default: def,
			},
			&answer,
		)
		return answer, err
	} else {
		return def, nil
	}
}

// PromptViewScript asks the user if they'd like to see a script,
// shows it if they answer yes, then asks if they'd still like to
// continue, and exits if they answer no.
func PromptViewScript(ctx context.Context, script, name, style string, interactive bool) error {
	if !interactive {
		return nil
	}

	view, err := YesNoPrompt(ctx, gotext.Get("Would you like to view the build script for %s", name), interactive, false)
	if err != nil {
		return err
	}

	if view {
		err = ShowScript(script, name, style)
		if err != nil {
			return err
		}

		cont, err := YesNoPrompt(ctx, gotext.Get("Would you still like to continue?"), interactive, false)
		if err != nil {
			return err
		}

		if !cont {
			slog.Error(gotext.Get("User chose not to continue after reading script"))
			os.Exit(1)
		}
	}

	return nil
}

// ShowScript uses the built-in pager to display a script at a
// given path, in the given syntax highlighting style.
func ShowScript(path, name, style string) error {
	scriptFl, err := os.Open(path)
	if err != nil {
		return err
	}
	defer scriptFl.Close()

	str, err := pager.SyntaxHighlightBash(scriptFl, style)
	if err != nil {
		return err
	}

	pgr := pager.New(name, str)
	return pgr.Run()
}

// FlattenPkgs attempts to flatten the a map of slices of packages into a single slice
// of packages by prompting the user if multiple packages match.
func FlattenPkgs(ctx context.Context, found map[string][]alrsh.Package, verb string, interactive bool) []alrsh.Package {
	return FlattenPkgsWithContext(ctx, found, verb, interactive, false)
}

// FlattenPkgsWithContext расширенная версия FlattenPkgs с контекстом обработки зависимостей
func FlattenPkgsWithContext(ctx context.Context, found map[string][]alrsh.Package, verb string, interactive bool, isDependency bool) []alrsh.Package {
	var outPkgs []alrsh.Package
	for _, pkgs := range found {
		if len(pkgs) > 1 {
			// Проверяем, являются ли пакеты подпакетами одного мультипакета
			if isMultiPackage(pkgs) && verb == "install" {
				// Для мультипакетов при установке ВСЕГДА берем все подпакеты без выбора
				// Это правильное поведение как для прямой установки, так и для зависимостей
				outPkgs = append(outPkgs, pkgs...)
			} else if interactive {
				// Для разных пакетов с одинаковым именем - показываем меню выбора
				choice, err := PkgPrompt(ctx, pkgs, verb, interactive)
				if err != nil {
					slog.Error(gotext.Get("Error prompting for choice of package"))
					os.Exit(1)
				}
				outPkgs = append(outPkgs, choice)
			} else {
				// Если не интерактивный режим - берем первый
				outPkgs = append(outPkgs, pkgs[0])
			}
		} else {
			// Если только один пакет - берем его
			outPkgs = append(outPkgs, pkgs[0])
		}
	}
	return outPkgs
}

// isMultiPackage проверяет, являются ли пакеты подпакетами одного мультипакета
func isMultiPackage(pkgs []alrsh.Package) bool {
	if len(pkgs) <= 1 {
		return false
	}
	
	// Проверяем, что у всех пакетов одинаковый BasePkgName и Repository
	firstBasePkg := pkgs[0].BasePkgName
	firstRepo := pkgs[0].Repository
	
	if firstBasePkg == "" {
		return false // Не мультипакет
	}
	
	for _, pkg := range pkgs[1:] {
		if pkg.BasePkgName != firstBasePkg || pkg.Repository != firstRepo {
			return false
		}
	}
	
	return true
}

// PkgPrompt asks the user to choose between multiple packages.
func PkgPrompt(ctx context.Context, options []alrsh.Package, verb string, interactive bool) (alrsh.Package, error) {
	if !interactive {
		return options[0], nil
	}

	names := make([]string, len(options))
	for i, option := range options {
		names[i] = option.Repository + "/" + option.Name + " " + option.Version
	}

	prompt := &survey.Select{
		Options: names,
		Message: gotext.Get("Choose which package to %s", verb),
	}

	var choice int
	err := survey.AskOne(prompt, &choice)
	if err != nil {
		return alrsh.Package{}, err
	}

	return options[choice], nil
}

// ChooseOptDepends asks the user to choose between multiple optional dependencies.
// The user may choose multiple items.
func ChooseOptDepends(ctx context.Context, options []string, verb string, interactive bool) ([]string, error) {
	if !interactive {
		return []string{}, nil
	}

	prompt := &survey.MultiSelect{
		Options: options,
		Message: gotext.Get("Choose which optional package(s) to install"),
	}

	var choices []int
	err := survey.AskOne(prompt, &choices)
	if err != nil {
		return nil, err
	}

	out := make([]string, len(choices))
	for i, choiceIndex := range choices {
		out[i], _, _ = strings.Cut(options[choiceIndex], ": ")
	}

	return out, nil
}
