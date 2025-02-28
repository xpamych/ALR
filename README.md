<p align="center">
    <img src="assets/logo.png" width="15%">
</p>
<b></b>

[![Go Report Card](https://goreportcard.com/badge/gitea.plemya-x.ru/Plemya-x/ALR)](https://goreportcard.com/report/gitea.plemya-x.ru/Plemya-x/ALR) ![Test coverage](./assets/coverage-badge.svg) ![ru translate](./assets/i18n-ru-badge.svg)

# ALR (Any Linux Repository)

ALR - это независимая от дистрибутива система сборки для Linux, аналогичная [AUR](https://wiki.archlinux.org/title/Arch_User_Repository). В настоящее время она находится в стадии бета-тестирования. Исправлено большинство основных ошибок и добавлено большинство важных функций. alr готов к общему использованию, но все еще может время от времени ломаться или заменяться.

ALR написан на чистом Go и после сборки не имеет зависимостей. Единственное, для повышения привилегий ALR требуется команда, такая как `sudo`, `doas` и т.д., а также поддерживаемый менеджер пакетов. В настоящее время ALR поддерживает `apt`, `pacman`, `apk`, `dnf`, `yum`, and `zypper`. Если в вашей системе существует поддерживаемый менеджер пакетов, он будет обнаружен и использован автоматически.

---

## Установка

### Установка скриптом

Установочный скрипт автоматически загрузит и установит соответствующий пакет ALR в вашей системе. Чтобы использовать его, просто выполните следующую команду:

```bash
curl -fsSL plemya-x.ru/alr/install.sh | bash
```

**ВАЖНО**: При этом скрипт будет загружен и запущен с <https://gitea.plemya-x.ru/Plemya-x/ALR/src/branch/master/scripts/install.sh>. Пожалуйста, просматривайте любые скрипты, которые вы скачиваете из Интернета (включая этот), прежде чем запускать их.

### Сборка из исходного кода

Чтобы собрать ALR из исходного кода, вам понадобится версия Go 1.18 или новее. Как только Go будет установлен, клонируйте это репозиторий и запустите:

```shell
make build
sudo make install
```

---

## Почему?

ALR был создан потому, что упаковка программного обеспечения для нескольких дистрибутивов Linux может быть сложной и чреватой ошибками, а установка этих пакетов может стать кошмаром для пользователей, если они не доступны в официальных репозиториях их дистрибутива. Он автоматизирует процесс создания и установки неофициальных пакетов.

---

## Документация

Документация по всем этим вопросам находится в [Wiki](https://gitea.plemya-x.ru/xpamych/ALR/wiki/Home).

---

## Репозитории

Репозитории alr - это git-хранилища, которые содержат каталог для каждого пакета с файлом `alr.sh` внутри. Файл `alr.sh` содержит все инструкции по сборке пакета и информацию о нем. Скрипты `alr.sh` аналогичны скриптам Aur PKGBUILD. 

Например, репозиторий [Plemya-x/xpamych-alr-repo](https://gitea.plemya-x.ru/Plemya-x/xpamych-alr-repo.git) можно подключить так:
```
alr addrepo --name xpamych-alr-repo --url https://gitea.plemya-x.ru/Plemya-x/xpamych-alr-repo.git
```

---
## Соцсети
VK - https://vk.com/plemya_kh

Discord - https://discord.com/channels/817759634105827358/1261631565084233749

Telegram - https://t.me/plemyakh

## Спасибы

Благодарим следующие проекты за то, что они сделали все возможное:

- <https://github.com/mvdan/sh>
- <https://github.com/go-git/go-git>
- <https://github.com/mholt/archiver>
- <https://github.com/goreleaser/nfpm>
- <https://github.com/charmbracelet/bubbletea>
- <https://gitlab.com/cznic/sqlite>
