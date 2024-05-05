<img src="assets/logo.png" alt="ALR Logo" width="200">

# ALR (Any Linux Repository)

ALR - это независимая от дистрибутива система сборки для Linux, аналогичная [AUR](https://wiki.archlinux.org/title/Arch_User_Repository). В настоящее время она находится в стадии бета-тестирования. Исправлено большинство основных ошибок и добавлено большинство важных функций. alr готов к общему использованию, но все еще может время от времени ломаться или заменяться.

alr написан на чистом Go и после сборки не имеет зависимостей. Единственное, для повышения привилегий alr требуется команда area, такая как "sudo", "doas" и т.д., а также поддерживаемый менеджер пакетов. В настоящее время alr поддерживает `apt`, `pacman`, `apk`, `dnf`, `yum`, and `zypper`. Если в вашей системе существует поддерживаемый менеджер пакетов, он будет обнаружен и использован автоматически.

---

## Установка

### Установка скриптом

Установочный скрипт автоматически загрузит и установит соответствующий пакет ALR в вашей системе. Чтобы использовать его, просто выполните следующую команду:

```bash
curl -fsSL plemya-x.ru/install | bash
```

**ВАЖНО**: При этом скрипт будет загружен и запущен с https://gitflic.ru/project/xpamych/alr/install. Пожалуйста, просматривайте любые скрипты, которые вы скачиваете из Интернета (включая этот), прежде чем запускать их.

### Пакеты

Пакеты для дистрибутивов и двоичные архивы представлены в последней версии на Gitflic: https://gitflic.ru/project/xpamych/alr/package

### Building from source

To build alr from source, you'll need Go 1.18 or newer. Once Go is installed, clone this repo and run:

```shell
sudo make install
```

---

## Why?

alr was created because packaging software for multiple Linux distros can be difficult and error-prone, and installing those packages can be a nightmare for users unless they're available in their distro's official repositories. It automates the process of building and installing unofficial packages.

---

## Documentation

The documentation for alr is in the [docs](docs) directory in this repo.

---

## Web Interface

alr has an open source web interface, licensed under the AGPLv3 (https://gitea.elara.ws/alr/alr-web), and it's available at https://gitflic.ru/project/xpamych/alr/.

---

## Repositories

alr's repos are git repositories that contain a directory for each package, with a `alr.sh` file inside. The `alr.sh` file tells alr how to build the package and information about it. `alr.sh` scripts are similar to the AUR's PKGBUILD scripts.

---

## Acknowledgements

Thanks to the following projects for making alr possible:

- https://github.com/mvdan/sh
- https://github.com/go-git/go-git
- https://github.com/mholt/archiver
- https://github.com/goreleaser/nfpm
- https://github.com/charmbracelet/bubbletea
- https://gitlab.com/cznic/sqlite