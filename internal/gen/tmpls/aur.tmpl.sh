# This file was originally part of the project "LURE - Linux User REpository", created by Elara Musayelyan.
# It has been modified as part of "ALR - Any Linux Repository" by the ALR Authors.
#
# ALR - Any Linux Repository
# Copyright (C) 2025 The ALR Authors
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.

# Generated from AUR package: {{.Name}}
# Package type: {{.PackageType}}
# AUR votes: {{.NumVotes}} | Popularity: {{printf "%.2f" .Popularity}}
# Original maintainer: {{.Maintainer}}
# Adapted for ALR by automation

name='{{.Name}}'
version='{{.Version}}'
release='1'
desc='{{.Description}}'
{{if ne .Description ""}}desc_ru='{{.Description}}'{{end}}
homepage='{{.URL}}'
maintainer="Евгений Храмов <xpamych@yandex.ru> (imported from AUR)"
{{if ne .Description ""}}maintainer_ru="Евгений Храмов <xpamych@yandex.ru> (импортирован из AUR)"{{end}}
architectures=({{.ArchitecturesString}})
license=({{.LicenseString}})
{{if .Provides}}provides=({{range .Provides}}'{{.}}' {{end}}){{end}}
{{if .Conflicts}}conflicts=({{range .Conflicts}}'{{.}}' {{end}}){{end}}
{{if .Replaces}}replaces=({{range .Replaces}}'{{.}}' {{end}}){{end}}

# Базовые зависимости
{{if .DependsString}}deps=({{.DependsString}}){{else}}deps=(){{end}}
{{if .MakeDependsString}}build_deps=({{.MakeDependsString}}){{else}}build_deps=(){{end}}

# Зависимости для конкретных дистрибутивов (адаптируйте под нужды пакета)
{{if .DependsString}}deps_arch=({{.DependsString}})
deps_debian=({{.DependsString}})
deps_altlinux=({{.DependsString}})
deps_alpine=({{.DependsString}}){{end}}

{{if and .MakeDependsString (ne .PackageType "bin")}}# Зависимости сборки для конкретных дистрибутивов
build_deps_arch=({{.MakeDependsString}})
build_deps_debian=({{.MakeDependsString}})
build_deps_altlinux=({{.MakeDependsString}})
build_deps_alpine=({{.MakeDependsString}}){{end}}

{{if .OptDependsString}}# Опциональные зависимости
opt_deps=(
	{{.OptDependsString}}
){{end}}

# Источники из PKGBUILD
sources=({{range .Sources}}"{{.}}" {{end}})
checksums=({{range .Checksums}}'{{.}}' {{end}})

{{if .HasVersion}}# Функция версии для Git-пакетов
version() {
	cd "$srcdir/{{.Name}}"
	git-version
}
{{end}}

{{if .ScriptsString}}# Дополнительные скрипты
scripts=(
	{{.ScriptsString}}
){{end}}

{{if or .PrepareFunc .HasPatches}}prepare() {
	cd "$srcdir"{{if .PrepareFunc}}
	# Из PKGBUILD:
	{{.PrepareFunc}}{{else}}
	# Применение патчей и подготовка исходников
	# Раскомментируйте и адаптируйте при необходимости:
	# patch -p1 < "${scriptdir}/fix.patch"{{end}}
}{{else}}# prepare() {
# 	cd "$srcdir"
# 	# Применение патчей и подготовка исходников при необходимости
# 	# patch -p1 < "${scriptdir}/fix.patch"
# }{{end}}

{{if ne .PackageType "bin"}}build() {
	cd "$srcdir"{{if .BuildFunc}}
	# Из PKGBUILD:
	{{.BuildFunc}}{{else}}
	
	# TODO: Адаптируйте команды сборки под конкретный проект ({{.PackageType}})
	{{if eq .PackageType "meson"}}# Для Meson проектов:
	meson setup build --prefix=/usr --buildtype=release
	ninja -C build -j $(nproc){{else if eq .PackageType "cpp"}}# Для C/C++ проектов:
	mkdir -p build && cd build
	cmake .. -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/usr
	make -j$(nproc){{else if eq .PackageType "go"}}# Для Go проектов:
	go build -buildmode=pie -trimpath -ldflags "-s -w" -o {{.Name}}{{else if eq .PackageType "python"}}# Для Python проектов:
	python -m build --wheel --no-isolation{{else if eq .PackageType "nodejs"}}# Для Node.js проектов:
	npm ci --production
	npm run build{{else if eq .PackageType "rust"}}# Для Rust проектов:
	cargo build --release --locked{{else if eq .PackageType "git"}}# Для Git проектов (обычно исходный код):
	make -j$(nproc){{else}}# Стандартная сборка:
	make -j$(nproc){{end}}{{end}}
}{{else}}# Бинарный пакет - сборка не требуется{{end}}

package() {
	cd "$srcdir"{{if .PackageFunc}}
	# Из PKGBUILD (адаптировано для ALR):
	{{.PackageFunc}}
	
	# Автоматически сгенерированные команды установки:
{{.GenerateInstallCommands}}{{else}}
	
	# TODO: Адаптируйте установку файлов под конкретный проект {{.Name}}
	{{if eq .PackageType "meson"}}# Для Meson проектов:
	meson install -C build --destdir="$pkgdir"{{else if eq .PackageType "cpp"}}# Для C/C++ проектов:
	cd build
	make DESTDIR="$pkgdir" install{{else if eq .PackageType "go"}}# Для Go проектов:
	# Исполняемый файл уже собран в корне{{else if eq .PackageType "python"}}# Для Python проектов:
	pip install --root="$pkgdir/" . --no-deps --disable-pip-version-check{{else if eq .PackageType "nodejs"}}# Для Node.js проектов:
	npm install -g --prefix="$pkgdir/usr" .{{else if eq .PackageType "rust"}}# Для Rust проектов:
	# Исполняемый файл в target/release/{{else if eq .PackageType "bin"}}# Бинарный пакет:
	# Файлы уже распакованы{{else}}# Стандартная установка:
	make DESTDIR="$pkgdir" install{{end}}
	
	# Автоматически сгенерированные команды установки:
{{.GenerateInstallCommands}}{{end}}
}