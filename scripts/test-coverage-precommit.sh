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

# Запускаем тесты с покрытием
make test-coverage

# coverage.out в .gitignore, не добавляем его
# Но если скрипт coverage-badge.sh изменил какие-то файлы (например, README с бейджем),
# они будут добавлены
CHANGED_FILES=$(git diff --name-only --diff-filter=M | grep -v '\.out$' | grep -v '^coverage' || true)

if [ ! -z "$CHANGED_FILES" ]; then
    echo "Test coverage updated the following files:"
    echo "$CHANGED_FILES"
    # Добавляем только измененные файлы, которые уже отслеживаются
    echo "$CHANGED_FILES" | xargs -r git add
    echo "Files were updated and staged"
fi

echo "Tests completed successfully"
# Всегда возвращаем успех если тесты прошли
exit 0