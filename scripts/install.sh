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

info() {
  echo $'\x1b[32m[ИНФО]\x1b[0m' $@
}

warn() {
  echo $'\x1b[31m[ВНИМАНИЕ]\x1b[0m' $@
}

error() {
  echo $'\x1b[31;1m[ОШИБКА]\x1b[0m' $@
  exit 1
}

installPkg() {
  rootCmd=""
  if command -v doas &>/dev/null; then
    rootCmd="doas"
  elif command -v sudo &>/dev/null; then
    rootCmd="sudo"
  else
    warn "Не обнаружена команда повышения привилегий (например, sudo, doas)"
  fi

  case $1 in
  pacman) $rootCmd pacman --noconfirm -U "${@:2}" ;;
  apk) $rootCmd apk add --allow-untrusted "${@:2}" ;;
  zypper) $rootCmd zypper --no-gpg-checks install "${@:2}" ;;
  *) $rootCmd "$1" install -y "${@:2}" ;;
  esac
}

if ! command -v curl &>/dev/null; then
  error "Этот скрипт требует команду curl. Пожалуйста, установите её и запустите снова."
fi

# Определение архитектуры системы
arch=$(uname -m)
case $arch in
  x86_64) debArch="amd64"; rpmArch="x86_64" ;;
  aarch64) debArch="arm64"; rpmArch="aarch64" ;;
  armv7l) debArch="armhf"; rpmArch="armv7hl" ;;
  *) error "Неподдерживаемая архитектура: $arch" ;;
esac

info "Обнаружена архитектура: $arch"

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
elif command -v apt-get &>/dev/null; then
  info "Обнаружен apt-get"
  pkgFormat="rpm"
  pkgMgr="apt-get"
else
  warn "Не обнаружен поддерживаемый менеджер пакетов!"
  noPkgMgr=true
fi

if [ -z "$noPkgMgr" ]; then
  info "Получение списка релизов через API Gitea"

  # Используем API для получения последнего релиза
  releases=$(curl -s "https://gitea.plemya-x.ru/api/v1/repos/Plemya-x/ALR/releases")

  if [ -z "$releases" ] || [ "$releases" = "null" ]; then
    error "Не удалось получить список релизов. Проверьте соединение с интернетом."
  fi

  # Получаем URL последнего релиза
  latestReleaseUrl=$(echo "$releases" | grep -o '"browser_download_url":"[^"]*"' | head -1 | cut -d'"' -f4)

  if [ -z "$latestReleaseUrl" ]; then
    # Fallback на парсинг HTML если API не работает
    warn "API не доступен, пробуем получить список через HTML"
    pageContent=$(curl -s https://gitea.plemya-x.ru/Plemya-x/ALR/releases)
    fileList=$(echo "$pageContent" | grep -oP '(?<=href=")[^"]*alr-bin[^"]*\.(pkg\.tar\.zst|rpm|deb)' | sed 's|^|https://gitea.plemya-x.ru|')
  else
    # Получаем список файлов из API
    latestReleaseId=$(echo "$releases" | grep -o '"id":[0-9]*' | head -1 | cut -d':' -f2)
    assets=$(curl -s "https://gitea.plemya-x.ru/api/v1/repos/Plemya-x/ALR/releases/$latestReleaseId/assets")
    # Фильтруем только пакеты, исключая tar.gz архивы
    fileList=$(echo "$assets" | grep -o '"browser_download_url":"[^"]*"' | cut -d'"' -f4 | grep -v '\.tar\.gz$')
  fi

  if [ -z "$fileList" ]; then
    warn "Не найдены готовые пакеты в последнем релизе"
    warn "Возможно, для вашего дистрибутива нужно собрать пакет из исходников"
    warn "Инструкции по сборке: https://gitea.plemya-x.ru/Plemya-x/ALR"
    error "Не удалось получить список пакетов для загрузки"
  fi

  info "Получен список файлов релиза"

  if [ "$pkgMgr" == "pacman" ]; then
      latestFile=$(echo "$fileList" | grep -E "alr-bin-.*\.pkg\.tar\.zst" | sort -V | tail -n 1)
  elif [ "$pkgMgr" == "apt" ]; then
      latestFile=$(echo "$fileList" | grep -E "alr-bin-.*\.${debArch}\.deb" | sort -V | tail -n 1)
  elif [[ "$pkgMgr" == "dnf" || "$pkgMgr" == "yum" || "$pkgMgr" == "zypper" ]]; then
      latestFile=$(echo "$fileList" | grep -E "alr-bin-.*\.${rpmArch}\.rpm" | grep -v 'alt[0-9]*' | sort -V | tail -n 1)
  elif [ "$pkgMgr" == "apt-get" ]; then
      # ALT Linux использует RPM с особой маркировкой
      latestFile=$(echo "$fileList" | grep -E "alr-bin-.*-alt[0-9]+\.${rpmArch}\.rpm" | sort -V | tail -n 1)
  elif [ "$pkgMgr" == "apk" ]; then
      latestFile=$(echo "$fileList" | grep -E "alr-bin-.*\.apk" | sort -V | tail -n 1)
  else
      error "Не поддерживаемый менеджер пакетов для автоматической установки"
  fi

  if [ -z "$latestFile" ]; then
      error "Не удалось найти соответствующий пакет для $pkgMgr"
  fi

  info "Найдена последняя версия ALR: $latestFile"

  fname="$(mktemp -u -p /tmp "alr.XXXXXXXXXX").${pkgFormat}"

  # Настраиваем trap для очистки временного файла
  trap "rm -f $fname" EXIT

  info "Загрузка пакета ALR"
  info "URL: $latestFile"

  # Загружаем с проверкой кода возврата
  if ! curl -f -L -o "$fname" "$latestFile"; then
      error "Ошибка загрузки пакета ALR. Проверьте подключение к интернету."
  fi

  # Проверяем что файл не пустой
  if [ ! -s "$fname" ]; then
      error "Загруженный файл пустой или поврежден"
  fi

  # Показываем размер загруженного файла
  fileSize=$(du -h "$fname" | cut -f1)
  info "Загружен пакет размером $fileSize"

  info "Установка пакета ALR"
  installPkg "$pkgMgr" "$fname"

  info "Очистка"
  rm -f "$fname"
  trap - EXIT

  info "Готово!"
else
  echo "Не найден поддерживаемый менеджер пакетов. О_о"
fi
