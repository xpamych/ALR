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

# Запускаем форматирование
make fmt || true

# Проверяем какие файлы были изменены (только те, что отслеживаются git)
CHANGED_FILES=$(git diff --name-only --diff-filter=M | grep '\.go$' || true)

# Если файлы были изменены, добавляем их в git
if [ ! -z "$CHANGED_FILES" ]; then
    echo "Formatting changed the following files:"
    echo "$CHANGED_FILES"
    # Добавляем только измененные файлы, которые уже отслеживаются
    echo "$CHANGED_FILES" | xargs -r git add
    echo "Files were formatted and staged"
fi

echo "Formatting completed"
# Всегда возвращаем успех
exit 0