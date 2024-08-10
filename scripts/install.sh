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
  pacman) $rootCmd pacman --noconfirm -U ${@:2} ;;
  apk) $rootCmd apk add --allow-untrusted ${@:2} ;;
  zypper) $rootCmd zypper --no-gpg-checks install ${@:2} ;;
  *) $rootCmd $1 install -y ${@:2} ;;
  esac
}

if ! command -v curl &>/dev/null; then
  error "Этот скрипт требует команду curl. Пожалуйста, установите её и запустите снова."
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
  warn "Не обнаружен поддерживаемый менеджер пакетов!"
  noPkgMgr=true
fi

if [ -z "$noPkgMgr" ]; then
  info "Получение списка файлов с https://plemya-x.ru/"
  pageContent=$(curl -s https://plemya-x.ru/?dir=alr)

  # Извлечение списка файлов из HTML
  fileList=$(echo "$pageContent" | grep -oP '(?<=href=").*?(?=")' | grep -E 'alr-bin-.*.(pkg.tar.zst|rpm|deb)')

  echo "Полученный список файлов:"
  echo "$fileList"

  if [ "$pkgMgr" == "pacman" ]; then
    latestFile=$(echo "$fileList" | grep -E 'alr-bin-.*.pkg.tar.zst' | sort -V | tail -n 1)
  elif [ "$pkgMgr" == "apt" ]; then
    latestFile=$(echo "$fileList" | grep -E 'alr-bin-.*.amd64.deb' | sort -V | tail -n 1)
  elif [[ "$pkgMgr" == "dnf" || "$pkgMgr" == "yum" || "$pkgMgr" == "zypper" ]]; then
    latestFile=$(echo "$fileList" | grep -E 'alr-bin-.*.x86_64.rpm' | sort -V | tail -n 1)
  else
    error "Не поддерживаемый менеджер пакетов для автоматической установки"
  fi

  if [ -z "$latestFile" ]; then
    error "Не удалось найти соответствующий пакет для $pkgMgr"
  fi

  info "Найдена последняя версия ALR: $latestFile"

  url="https://plemya-x.ru/$latestFile"
  fname="$(mktemp -u -p /tmp "alr.XXXXXXXXXX").${pkgFormat}"

  info "Загрузка пакета ALR"
  curl -L $url -o $fname

  if [ ! -f "$fname" ]; then
    error "Ошибка загрузки пакета ALR"
  fi

  info "Установка пакета ALR"
  installPkg $pkgMgr $fname

  info "Очистка"
  rm $fname

  info "Готово!"
else
  info "Клонирование репозитория ALR"
  git clone https://gitverse.ru/sc/Xpamych/ALR.git /tmp/alr

  info "Установка ALR"
  cd /tmp/alr
  sudo make install

  info "Очистка репозитория ALR"
  rm -rf /tmp/alr

  info "Все задачи выполнены!"
fi
