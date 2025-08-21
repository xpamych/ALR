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

package gen

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"text/template"
)

// Встраиваем шаблон для AUR пакетов
//
//go:embed tmpls/aur.tmpl.sh
var aurTmpl string

// AUROptions содержит параметры для генерации шаблона AUR
type AUROptions struct {
	Name    string // Имя пакета в AUR
	Version string // Версия пакета (опционально, если не указана - берется последняя)
	CreateDir bool  // Создавать ли директорию для пакета и дополнительные файлы
}

// aurAPIResponse представляет структуру ответа от API AUR
type aurAPIResponse struct {
	Version      int         `json:"version"`      // Версия API
	Type         string      `json:"type"`         // Тип ответа
	ResultCount  int         `json:"resultcount"`  // Количество результатов
	Results      []aurResult `json:"results"`      // Массив результатов
	Error        string      `json:"error"`        // Сообщение об ошибке (если есть)
}

// aurResult содержит информацию о пакете из AUR
type aurResult struct {
	ID             int      `json:"ID"`
	Name           string   `json:"Name"`
	PackageBaseID  int      `json:"PackageBaseID"`
	PackageBase    string   `json:"PackageBase"`
	Version        string   `json:"Version"`
	Description    string   `json:"Description"`
	URL            string   `json:"URL"`
	NumVotes       int      `json:"NumVotes"`
	Popularity     float64  `json:"Popularity"`
	OutOfDate      *int     `json:"OutOfDate"`
	Maintainer     string   `json:"Maintainer"`
	FirstSubmitted int      `json:"FirstSubmitted"`
	LastModified   int      `json:"LastModified"`
	URLPath        string   `json:"URLPath"`
	License        []string `json:"License"`
	Keywords       []string `json:"Keywords"`
	Depends        []string `json:"Depends"`
	MakeDepends    []string `json:"MakeDepends"`
	OptDepends     []string `json:"OptDepends"`
	CheckDepends   []string `json:"CheckDepends"`
	Conflicts      []string `json:"Conflicts"`
	Provides       []string `json:"Provides"`
	Replaces       []string `json:"Replaces"`
	// Дополнительные поля для данных из PKGBUILD
	Sources      []string `json:"-"`
	Checksums    []string `json:"-"`
	BuildFunc    string   `json:"-"`
	PackageFunc  string   `json:"-"`
	PrepareFunc  string   `json:"-"`
	PackageType  string   `json:"-"`  // python, go, rust, cpp, nodejs, bin, git
	HasDesktop   bool     `json:"-"`  // Есть ли desktop файлы
	HasSystemd   bool     `json:"-"`  // Есть ли systemd сервисы
	HasVersion   bool     `json:"-"`  // Есть ли функция version()
	HasScripts   []string `json:"-"`  // Дополнительные скрипты (postinstall, postremove, etc)
	HasPatches   bool     `json:"-"`  // Есть ли патчи
	Architectures []string `json:"-"` // Поддерживаемые архитектуры
	
	// Автоматически определяемые файлы для install-* команд
	BinaryFiles  []string `json:"-"`  // Исполняемые файлы для install-binary
	LicenseFiles []string `json:"-"`  // Лицензионные файлы для install-license
	ManualFiles  []string `json:"-"`  // Man страницы для install-manual
	DesktopFiles []string `json:"-"`  // Desktop файлы для install-desktop
	ServiceFiles []string `json:"-"`  // Systemd сервисы для install-systemd
	CompletionFiles map[string]string `json:"-"` // Файлы автодополнения по типу (bash, zsh, fish)
}

// Вспомогательные методы для шаблона
func (r aurResult) LicenseString() string {
	if len(r.License) == 0 {
		return "custom:Unknown"
	}
	// Форматируем лицензии для alr.sh
	licenses := make([]string, len(r.License))
	for i, l := range r.License {
		licenses[i] = fmt.Sprintf("'%s'", l)
	}
	return strings.Join(licenses, " ")
}

func (r aurResult) DependsString() string {
	if len(r.Depends) == 0 {
		return ""
	}
	deps := make([]string, len(r.Depends))
	for i, d := range r.Depends {
		// Убираем версионные ограничения для простоты
		dep := strings.Split(d, ">=")[0]
		dep = strings.Split(dep, "<=")[0]
		dep = strings.Split(dep, "=")[0]
		dep = strings.Split(dep, ">")[0]
		dep = strings.Split(dep, "<")[0]
		deps[i] = fmt.Sprintf("'%s'", dep)
	}
	return strings.Join(deps, " ")
}

func (r aurResult) MakeDependsString() string {
	if len(r.MakeDepends) == 0 {
		return ""
	}
	deps := make([]string, len(r.MakeDepends))
	for i, d := range r.MakeDepends {
		// Убираем версионные ограничения для простоты
		dep := strings.Split(d, ">=")[0]
		dep = strings.Split(dep, "<=")[0]
		dep = strings.Split(dep, "=")[0]
		dep = strings.Split(dep, ">")[0]
		dep = strings.Split(dep, "<")[0]
		deps[i] = fmt.Sprintf("'%s'", dep)
	}
	return strings.Join(deps, " ")
}

func (r aurResult) GitURL() string {
	// Формируем URL для клонирования из AUR
	return fmt.Sprintf("https://aur.archlinux.org/%s.git", r.PackageBase)
}

func (r aurResult) ArchitecturesString() string {
	if len(r.Architectures) == 0 {
		return "'all'"
	}
	archs := make([]string, len(r.Architectures))
	for i, arch := range r.Architectures {
		archs[i] = fmt.Sprintf("'%s'", arch)
	}
	return strings.Join(archs, " ")
}

func (r aurResult) OptDependsString() string {
	if len(r.OptDepends) == 0 {
		return ""
	}
	optDeps := make([]string, 0, len(r.OptDepends))
	for _, dep := range r.OptDepends {
		// Форматируем опциональные зависимости для alr.sh
		parts := strings.SplitN(dep, ": ", 2)
		if len(parts) == 2 {
			optDeps = append(optDeps, fmt.Sprintf("'%s: %s'", parts[0], parts[1]))
		} else {
			optDeps = append(optDeps, fmt.Sprintf("'%s'", dep))
		}
	}
	return strings.Join(optDeps, "\n\t")
}

func (r aurResult) ScriptsString() string {
	if len(r.HasScripts) == 0 {
		return ""
	}
	scripts := make([]string, len(r.HasScripts))
	for i, script := range r.HasScripts {
		scripts[i] = fmt.Sprintf("['%s']='%s.sh'", script, script)
	}
	return strings.Join(scripts, "\n\t")
}

// GenerateInstallCommands генерирует команды install-* для шаблона
func (r aurResult) GenerateInstallCommands() string {
	var commands []string
	
	// install-binary команды
	for _, binary := range r.BinaryFiles {
		if binary == "./"+r.Name {
			commands = append(commands, fmt.Sprintf("\tinstall-binary %s", binary))
		} else {
			commands = append(commands, fmt.Sprintf("\tinstall-binary %s %s", binary, r.Name))
		}
	}
	
	// install-license команды
	for _, license := range r.LicenseFiles {
		if license == "LICENSE" || license == "./LICENSE" {
			commands = append(commands, fmt.Sprintf("\tinstall-license %s %s/LICENSE", license, r.Name))
		} else {
			commands = append(commands, fmt.Sprintf("\tinstall-license %s %s/LICENSE", license, r.Name))
		}
	}
	
	// install-manual команды
	for _, manual := range r.ManualFiles {
		commands = append(commands, fmt.Sprintf("\tinstall-manual %s", manual))
	}
	
	// install-desktop команды
	for _, desktop := range r.DesktopFiles {
		commands = append(commands, fmt.Sprintf("\tinstall-desktop %s", desktop))
	}
	
	// install-systemd команды
	for _, service := range r.ServiceFiles {
		if strings.Contains(service, "user") {
			commands = append(commands, fmt.Sprintf("\tinstall-systemd-user %s", service))
		} else {
			commands = append(commands, fmt.Sprintf("\tinstall-systemd %s", service))
		}
	}
	
	// install-completion команды
	for shell, file := range r.CompletionFiles {
		switch shell {
		case "bash":
			commands = append(commands, fmt.Sprintf("\tinstall-completion bash %s < %s", r.Name, file))
		case "zsh":
			commands = append(commands, fmt.Sprintf("\tinstall-completion zsh %s < %s", r.Name, file))
		case "fish":
			commands = append(commands, fmt.Sprintf("\t%s completion fish | install-completion fish %s", r.Name, r.Name))
		}
	}
	
	if len(commands) == 0 {
		return "\t# TODO: Добавьте команды установки файлов"
	}
	
	return strings.Join(commands, "\n")
}

// fetchPKGBUILD загружает PKGBUILD файл для пакета
func fetchPKGBUILD(packageBase string) (string, error) {
	// URL для raw PKGBUILD
	pkgbuildURL := fmt.Sprintf("https://aur.archlinux.org/cgit/aur.git/plain/PKGBUILD?h=%s", packageBase)
	
	res, err := http.Get(pkgbuildURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch PKGBUILD: %w", err)
	}
	defer res.Body.Close()
	
	if res.StatusCode != 200 {
		return "", fmt.Errorf("failed to fetch PKGBUILD: status %s", res.Status)
	}
	
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read PKGBUILD: %w", err)
	}
	
	return string(data), nil
}

// parseSources извлекает источники из PKGBUILD
func parseSources(pkgbuild string) []string {
	var sources []string
	
	// Регулярное выражение для поиска массива source
	// Поддерживает как однострочные, так и многострочные определения
	sourceRegex := regexp.MustCompile(`(?ms)source=\((.*?)\)`)
	matches := sourceRegex.FindStringSubmatch(pkgbuild)
	
	if len(matches) > 1 {
		// Извлекаем содержимое массива source
		sourceContent := matches[1]
		
		// Разбираем элементы массива
		// Учитываем кавычки и переносы строк
		elemRegex := regexp.MustCompile(`['"]([^'"]+)['"]`)
		elements := elemRegex.FindAllStringSubmatch(sourceContent, -1)
		
		for _, elem := range elements {
			if len(elem) > 1 {
				source := elem[1]
				// Заменяем переменные версии
				source = strings.ReplaceAll(source, "$pkgver", "${version}")
				source = strings.ReplaceAll(source, "${pkgver}", "${version}")
				source = strings.ReplaceAll(source, "$pkgname", "${name}")
				source = strings.ReplaceAll(source, "${pkgname}", "${name}")
				// Обрабатываем другие переменные (упрощенно)
				source = strings.ReplaceAll(source, "$_commit", "${_commit}")
				source = strings.ReplaceAll(source, "${_commit}", "${_commit}")
				sources = append(sources, source)
			}
		}
	}
	
	// Если источники не найдены в source=(), проверяем source_x86_64 и другие архитектуры
	if len(sources) == 0 {
		archSourceRegex := regexp.MustCompile(`(?ms)source_(?:x86_64|aarch64)=\((.*?)\)`)
		matches = archSourceRegex.FindStringSubmatch(pkgbuild)
		if len(matches) > 1 {
			sourceContent := matches[1]
			elemRegex := regexp.MustCompile(`['"]([^'"]+)['"]`)
			elements := elemRegex.FindAllStringSubmatch(sourceContent, -1)
			
			for _, elem := range elements {
				if len(elem) > 1 {
					source := elem[1]
					source = strings.ReplaceAll(source, "$pkgver", "${version}")
					source = strings.ReplaceAll(source, "${pkgver}", "${version}")
					source = strings.ReplaceAll(source, "$pkgname", "${name}")
					source = strings.ReplaceAll(source, "${pkgname}", "${name}")
					sources = append(sources, source)
				}
			}
		}
	}
	
	return sources
}

// parseChecksums извлекает контрольные суммы из PKGBUILD
func parseChecksums(pkgbuild string) []string {
	var checksums []string
	
	// Пробуем разные типы контрольных сумм
	for _, hashType := range []string{"sha256sums", "sha512sums", "sha1sums", "md5sums", "b2sums"} {
		regex := regexp.MustCompile(fmt.Sprintf(`(?ms)%s=\((.*?)\)`, hashType))
		matches := regex.FindStringSubmatch(pkgbuild)
		
		if len(matches) > 1 {
			content := matches[1]
			elemRegex := regexp.MustCompile(`['"]([^'"]+)['"]`)
			elements := elemRegex.FindAllStringSubmatch(content, -1)
			
			for _, elem := range elements {
				if len(elem) > 1 {
					checksums = append(checksums, elem[1])
				}
			}
			
			if len(checksums) > 0 {
				break // Используем первый найденный тип хешей
			}
		}
	}
	
	return checksums
}

// parseFunctions извлекает функции build(), package() и prepare() из PKGBUILD
func parseFunctions(pkgbuild string) (buildFunc, packageFunc, prepareFunc string) {
	// Извлекаем функцию build()
	buildRegex := regexp.MustCompile(`(?ms)^build\(\)\s*\{(.*?)^\}`)
	if matches := buildRegex.FindStringSubmatch(pkgbuild); len(matches) > 1 {
		buildFunc = strings.TrimSpace(matches[1])
	}
	
	// Извлекаем функцию package()
	packageRegex := regexp.MustCompile(`(?ms)^package\(\)\s*\{(.*?)^\}`)
	if matches := packageRegex.FindStringSubmatch(pkgbuild); len(matches) > 1 {
		packageFunc = strings.TrimSpace(matches[1])
	}
	
	// Извлекаем функцию prepare()
	prepareRegex := regexp.MustCompile(`(?ms)^prepare\(\)\s*\{(.*?)^\}`)
	if matches := prepareRegex.FindStringSubmatch(pkgbuild); len(matches) > 1 {
		prepareFunc = strings.TrimSpace(matches[1])
	}
	
	return buildFunc, packageFunc, prepareFunc
}

// detectInstallableFiles анализирует PKGBUILD и определяет файлы для install-* команд
func detectInstallableFiles(pkg *aurResult, pkgbuild string) {
	// Инициализируем карту для файлов автодополнения
	pkg.CompletionFiles = make(map[string]string)
	
	// Для простоты, добавляем стандартные файлы для типа пакета
	switch pkg.PackageType {
	case "go":
		pkg.BinaryFiles = append(pkg.BinaryFiles, "./"+pkg.Name)
	case "rust":
		pkg.BinaryFiles = append(pkg.BinaryFiles, "./target/release/"+pkg.Name)
	case "cpp", "meson":
		pkg.BinaryFiles = append(pkg.BinaryFiles, "./"+pkg.Name) // обычно в корне после сборки
	case "bin":
		pkg.BinaryFiles = append(pkg.BinaryFiles, "./"+pkg.Name)
	default:
		if pkg.PackageType != "python" && pkg.PackageType != "nodejs" {
			pkg.BinaryFiles = append(pkg.BinaryFiles, "./"+pkg.Name)
		}
	}
	
	// Ищем лицензионные файлы для install-license с более точными паттернами
	licenseRegex := regexp.MustCompile(`(?i)\b(LICENSE|COPYING|COPYRIGHT|LICENCE)(?:\.[a-zA-Z0-9]+)?\b`)
	licenseMatches := licenseRegex.FindAllString(pkgbuild, -1)
	for _, match := range licenseMatches {
		// Фильтруем только реальные файлы лицензий
		if strings.Contains(strings.ToLower(match), "license") || 
		   strings.Contains(strings.ToLower(match), "copying") || 
		   strings.Contains(strings.ToLower(match), "copyright") {
			if !contains(pkg.LicenseFiles, "./"+match) {
				pkg.LicenseFiles = append(pkg.LicenseFiles, "./"+match)
			}
		}
	}
	
	// Если не найдены лицензионные файлы, добавляем стандартные
	if len(pkg.LicenseFiles) == 0 {
		pkg.LicenseFiles = append(pkg.LicenseFiles, "LICENSE")
	}
	
	// Ищем man страницы для install-manual с более точными паттернами
	manRegex := regexp.MustCompile(`\b\w+\.(?:1|2|3|4|5|6|7|8)(?:\.gz)?\b`)
	manMatches := manRegex.FindAllString(pkgbuild, -1)
	for _, match := range manMatches {
		// Проверяем, что это не переменная или часть кода
		if !strings.Contains(match, "$") && !strings.Contains(match, "{") {
			if !contains(pkg.ManualFiles, "./"+match) {
				pkg.ManualFiles = append(pkg.ManualFiles, "./"+match)
			}
		}
	}
	
	// Ищем desktop файлы для install-desktop
	desktopRegex := regexp.MustCompile(`[^/\s]*\.desktop`)
	desktopMatches := desktopRegex.FindAllString(pkgbuild, -1)
	for _, match := range desktopMatches {
		if !contains(pkg.DesktopFiles, "./"+match) {
			pkg.DesktopFiles = append(pkg.DesktopFiles, "./"+match)
		}
	}
	
	// Ищем systemd сервисы для install-systemd
	serviceRegex := regexp.MustCompile(`[^/\s]*\.service`)
	serviceMatches := serviceRegex.FindAllString(pkgbuild, -1)
	for _, match := range serviceMatches {
		if !contains(pkg.ServiceFiles, "./"+match) {
			pkg.ServiceFiles = append(pkg.ServiceFiles, "./"+match)
		}
	}
	
	// Ищем файлы автодополнения
	completionPatterns := map[string]string{
		"bash": `completions?/.*\.bash|bash-completion`,
		"zsh":  `completions?/.*\.zsh|zsh.*completion`,
		"fish": `completions?/.*\.fish|fish.*completion`,
	}
	
	for shell, pattern := range completionPatterns {
		regex := regexp.MustCompile(fmt.Sprintf(`(?i)%s`, pattern))
		matches := regex.FindAllString(pkgbuild, -1)
		if len(matches) > 0 {
			pkg.CompletionFiles[shell] = matches[0]
		}
	}
}

// contains проверяет, содержит ли слайс строк указанную строку
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// detectPackageType определяет тип пакета на основе имени, зависимостей и источников
func detectPackageType(pkg *aurResult, pkgbuild string) {
	name := strings.ToLower(pkg.Name)
	
	// Определяем тип на основе имени пакета
	switch {
	case strings.HasPrefix(name, "python") || strings.HasPrefix(name, "python3-"):
		pkg.PackageType = "python"
	case strings.Contains(name, "nodejs") || strings.Contains(name, "node-"):
		pkg.PackageType = "nodejs"
	case strings.HasSuffix(name, "-bin"):
		pkg.PackageType = "bin"
	case strings.HasSuffix(name, "-git"):
		pkg.PackageType = "git"
		pkg.HasVersion = true // Git пакеты обычно имеют функцию version()
	case strings.Contains(name, "rust") || hasRustSources(pkg.Sources):
		pkg.PackageType = "rust"
	case strings.Contains(name, "go-") || hasGoSources(pkg.Sources):
		pkg.PackageType = "go"
	case strings.Contains(name, "-rust") || strings.Contains(name, "paru") || strings.Contains(name, "cargo-"):
		pkg.PackageType = "rust"
	default:
		// Определяем по зависимостям сборки
		for _, dep := range pkg.MakeDepends {
			depLower := strings.ToLower(dep)
			switch {
			case strings.Contains(depLower, "meson") || strings.Contains(depLower, "ninja"):
				pkg.PackageType = "meson"
			case strings.Contains(depLower, "cmake") || strings.Contains(depLower, "gcc") || strings.Contains(depLower, "clang"):
				pkg.PackageType = "cpp"
			case strings.Contains(depLower, "python"):
				pkg.PackageType = "python"
			case strings.Contains(depLower, "go"):
				pkg.PackageType = "go"
			case strings.Contains(depLower, "rust") || strings.Contains(depLower, "cargo"):
				pkg.PackageType = "rust"
			case strings.Contains(depLower, "npm") || strings.Contains(depLower, "nodejs"):
				pkg.PackageType = "nodejs"
			}
			if pkg.PackageType != "" {
				break
			}
		}
	}
	
	// Определяем архитектуры на основе типа пакета
	if pkg.PackageType == "bin" {
		pkg.Architectures = []string{"amd64"} // Бинарные пакеты обычно специфичны для архитектуры
	} else {
		pkg.Architectures = []string{"all"} // Исходный код собирается для любой архитектуры
	}
	
	// Определяем наличие desktop файлов
	pkg.HasDesktop = strings.Contains(pkgbuild, ".desktop") || 
		strings.Contains(pkgbuild, "install-desktop") ||
		strings.Contains(pkgbuild, "xdg-desktop")
	
	// Определяем наличие systemd сервисов
	pkg.HasSystemd = strings.Contains(pkgbuild, ".service") ||
		strings.Contains(pkgbuild, "systemctl") ||
		strings.Contains(pkgbuild, "install-systemd")
	
	// Определяем наличие функции version() для -git пакетов
	pkg.HasVersion = strings.Contains(pkgbuild, "pkgver()") || 
		(strings.HasSuffix(name, "-git") && strings.Contains(pkgbuild, "git describe"))
	
	// Определяем наличие патчей
	pkg.HasPatches = strings.Contains(pkgbuild, "patch ") || 
		strings.Contains(pkgbuild, ".patch") ||
		strings.Contains(pkgbuild, ".diff")
	
	// Определяем дополнительные скрипты
	if strings.Contains(pkgbuild, "post_install") {
		pkg.HasScripts = append(pkg.HasScripts, "postinstall")
	}
	if strings.Contains(pkgbuild, "pre_remove") || strings.Contains(pkgbuild, "post_remove") {
		pkg.HasScripts = append(pkg.HasScripts, "postremove")
	}
}

// hasRustSources проверяет, содержат ли источники Rust проекты
func hasRustSources(sources []string) bool {
	for _, src := range sources {
		if strings.Contains(src, "crates.io") || strings.Contains(src, "Cargo.toml") {
			return true
		}
	}
	return false
}

// hasGoSources проверяет, содержат ли источники Go проекты
func hasGoSources(sources []string) bool {
	for _, src := range sources {
		if strings.Contains(src, "github.com") && strings.Contains(src, "/go") {
			return true
		}
	}
	return false
}

// AUR генерирует шаблон alr.sh на основе пакета из AUR
func AUR(w io.Writer, opts AUROptions) error {
	// Создаем шаблон с функциями
	tmpl, err := template.New("aur").
		Funcs(funcs).
		Parse(aurTmpl)
	if err != nil {
		return err
	}

	// Формируем URL запроса к AUR API
	apiURL := "https://aur.archlinux.org/rpc/v5/info"
	params := url.Values{}
	params.Add("arg[]", opts.Name)
	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	// Выполняем запрос к AUR API
	res, err := http.Get(fullURL)
	if err != nil {
		return fmt.Errorf("failed to fetch AUR package info: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("AUR API returned status: %s", res.Status)
	}

	// Декодируем ответ
	var resp aurAPIResponse
	err = json.NewDecoder(res.Body).Decode(&resp)
	if err != nil {
		return fmt.Errorf("failed to decode AUR response: %w", err)
	}

	// Проверяем наличие ошибки в ответе
	if resp.Error != "" {
		return fmt.Errorf("AUR API error: %s", resp.Error)
	}

	// Проверяем, что пакет найден
	if resp.ResultCount == 0 {
		return fmt.Errorf("package '%s' not found in AUR", opts.Name)
	}

	// Берем первый результат
	pkg := resp.Results[0]

	// Если указана версия, проверяем соответствие
	if opts.Version != "" && pkg.Version != opts.Version {
		// Предупреждаем, но продолжаем с актуальной версией из AUR
		fmt.Fprintf(w, "# WARNING: Requested version %s, but AUR has %s\n", opts.Version, pkg.Version)
	}

	// Загружаем PKGBUILD для получения источников
	pkgbuild, err := fetchPKGBUILD(pkg.PackageBase)
	if err != nil {
		// Если не удалось загрузить PKGBUILD, используем fallback на AUR репозиторий
		fmt.Fprintf(w, "# WARNING: Could not fetch PKGBUILD: %v\n", err)
		fmt.Fprintf(w, "# Using AUR repository as source\n")
		pkg.Sources = []string{fmt.Sprintf("%s::git+%s", pkg.Name, pkg.GitURL())}
		pkg.Checksums = []string{"SKIP"}
	} else {
		// Извлекаем источники из PKGBUILD
		pkg.Sources = parseSources(pkgbuild)
		pkg.Checksums = parseChecksums(pkgbuild)
		pkg.BuildFunc, pkg.PackageFunc, pkg.PrepareFunc = parseFunctions(pkgbuild)
		
		// Определяем тип пакета
		detectPackageType(&pkg, pkgbuild)
		
		// Определяем файлы для install-* команд
		detectInstallableFiles(&pkg, pkgbuild)
		
		// Если источники не найдены, используем fallback
		if len(pkg.Sources) == 0 {
			fmt.Fprintf(w, "# WARNING: No sources found in PKGBUILD\n")
			fmt.Fprintf(w, "# Using AUR repository as source\n")
			pkg.Sources = []string{fmt.Sprintf("%s::git+%s", pkg.Name, pkg.GitURL())}
			pkg.Checksums = []string{"SKIP"}
		}
	}

	// Выполняем шаблон
	return tmpl.Execute(w, pkg)
}