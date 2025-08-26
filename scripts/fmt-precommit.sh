#!/bin/bash
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

set -e

# Сохраняем хеши файлов до форматирования
TEMP_DIR=$(mktemp -d)
find . -name "*.go" -type f | while read file; do
    if [ -f "$file" ]; then
        md5sum "$file" > "$TEMP_DIR/$(basename $file).md5" 2>/dev/null || true
    fi
done

# Запускаем форматирование
make fmt || true

# Проверяем, были ли изменения
CHANGED=false
find . -name "*.go" -type f | while read file; do
    if [ -f "$file" ] && [ -f "$TEMP_DIR/$(basename $file).md5" ]; then
        OLD_MD5=$(cat "$TEMP_DIR/$(basename $file).md5" | awk '{print $1}')
        NEW_MD5=$(md5sum "$file" | awk '{print $1}')
        if [ "$OLD_MD5" != "$NEW_MD5" ]; then
            CHANGED=true
            break
        fi
    fi
done

# Удаляем временную директорию
rm -rf "$TEMP_DIR"

# Если файлы были изменены, добавляем их в git
if [ "$CHANGED" = true ]; then
    git add -u
    echo "Files were formatted and staged"
fi

# Всегда возвращаем успех
exit 0