#!/bin/bash

# ALR - Any Linux Repository
# Copyright (C) 2024 Евгений Храмов
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

info() {
  echo $'\x1b[32m[INFO]\x1b[0m' $@
}

warn() {
  echo $'\x1b[31m[WARN]\x1b[0m' $@
}

error() {
  echo $'\x1b[31;1m[ERR]\x1b[0m' $@
  exit 1
}

installPkg() {
  rootCmd=""
  if command -v doas &>/dev/null; then
    rootCmd="doas"
  elif command -v sudo &>/dev/null; then
    rootCmd="sudo"
  else
    warn "Команда повышения привилегий (например, sudo, do as) не обнаружена"
  fi
  
  case $1 in
  pacman) $rootCmd pacman --noconfirm -U ${@:2} ;;
  apk) $rootCmd apk add --allow-untrusted ${@:2} ;;
  zypper) $rootCmd zypper --no-gpg-checks install ${@:2} ;;
  *) $rootCmd $1 install -y ${@:2} ;;
  esac
}

if ! command -v curl &>/dev/null; then
  error "Для этого скрипта требуется команда curl. Пожалуйста, установите его и запустите снова."
fi

pkgFormat=""
pkgMgr=""
if command -v pacman &>/dev/null; then
  info "Обнаружен pacman"
  pkgFormat="pkg.tar.zst"
  pkgMgr="pacman"
elif command -v apt &>/dev/null; then
  info "Обнаружен apt"
  pkgFormat="deb"
  pkgMgr="apt"
elif command -v dnf &>/dev/null; then
  info "Обнаружен dnf"
  pkgFormat="rpm"
  pkgMgr="dnf"
elif command -v yum &>/dev/null; then
  info "Обнаружен yum"
  pkgFormat="rpm"
  pkgMgr="yum"
elif command -v zypper &>/dev/null; then
  info "Обнаружен zypper"
  pkgFormat="rpm"
  pkgMgr="zypper"
elif command -v apk &>/dev/null; then
  info "Обнаружен apk"
  pkgFormat="apk"
  pkgMgr="apk"
else
  error "Не обнаружен поддерживаемый пакетный менеджер!"
fi

# Заменить на запрос версии через api gitflic
#latestVersion=$(curl -sI 'https://gitflic.ru/project/xpamych/alr/release/latest' | grep -io 'location: .*' | rev | cut -d '/' -f1 | rev | tr -d '[:space:]')
#info "Найдена последняя версия ALR:" $latestVersion

fname="$(mktemp -u -p /tmp "alr.XXXXXXXXXX").${pkgFormat}"
url="https://registry.gitflic.ru/project/xpamych/alr/package/-/generic/alr-linux-x86-64/${latestVersion}/releases-${latestVersion}.${pkgFormat}"

info "Скачивается пакет ALR"
curl --location --request GET $url -o $fname

info "Устанавливается ALR"
installPkg $pkgMgr $fname

info "Очистка"
rm $fname

info "Готово!"