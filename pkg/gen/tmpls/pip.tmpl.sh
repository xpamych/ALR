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

name='python3-{{.Info.Name | tolower}}'
version='{{.Info.Version}}'
release='1'
desc='{{.Info.Summary}}'
homepage='{{.Info.Homepage}}'
maintainer='Example <user@example.com>'
architectures=('all')
license=('{{if .Info.License | ne ""}}{{.Info.License}}{{else}}custom:Unknown{{end}}')
provides=('{{.Info.Name | tolower}}')
conflicts=('{{.Info.Name | tolower}}')

deps=("python3")
deps_arch=("python")
deps_alpine=("python3")

build_deps=("python3" "python3-pip")
build_deps_arch=("python" "python-pip")
build_deps_alpine=("python3" "py3-pip")

sources=("https://files.pythonhosted.org/packages/source/{{.SourceURL.Filename | firstchar}}/{{.Info.Name}}/{{.SourceURL.Filename}}")
checksums=('blake2b-256:{{.SourceURL.Digests.blake2b_256}}')

build() {
	cd "$srcdir/{{.Info.Name}}-${version}"
  python -m build --wheel --no-isolation
}

package() {
	cd "$srcdir/{{.Info.Name}}-${version}"
	pip install --root="${pkgdir}/" . --no-deps --ignore-installed --disable-pip-version-check
}

files() {
  printf '"%s" ' ./usr/local/lib/python3.*/site-packages/{{.Info.Name | tolower}}/*
  printf '"%s" ' ./usr/local/lib/python3.*/site-packages/{{.Info.Name | tolower}}-${version}.dist-info/*
}