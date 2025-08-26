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

# Wrapper script for i18n that automatically stages changed files for pre-commit

set -e

# Сохраняем состояние файлов до выполнения i18n
TRANSLATION_FILES=(
    "internal/translations/default.pot"
    "internal/translations/po/ru/default.po"
    "assets/i18n-ru-badge.svg"
)

# Создаем временные файлы для сравнения
TEMP_DIR=$(mktemp -d)
for file in "${TRANSLATION_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        cp "$file" "$TEMP_DIR/$(basename "$file")"
    fi
done

# Выполняем обновление переводов
make i18n

# Проверяем какие файлы изменились и добавляем их в staging area
CHANGED_FILES=()
for file in "${TRANSLATION_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        if [[ ! -f "$TEMP_DIR/$(basename "$file")" ]] || ! cmp -s "$file" "$TEMP_DIR/$(basename "$file")"; then
            CHANGED_FILES+=("$file")
        fi
    fi
done

# Добавляем измененные файлы в git staging area
if [[ ${#CHANGED_FILES[@]} -gt 0 ]]; then
    echo "Auto-staging changed translation files:"
    for file in "${CHANGED_FILES[@]}"; do
        echo "  - $file"
        git add "$file"
    done
fi

# Очищаем временные файлы
rm -rf "$TEMP_DIR"

# Выход с кодом 0 (успех) даже если файлы были изменены
exit 0