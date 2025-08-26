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

# Если coverage.out был изменен, добавляем его
if git diff --quiet coverage.out 2>/dev/null; then
    echo "Coverage unchanged"
else
    git add coverage.out 2>/dev/null || true
    echo "Coverage updated and staged"
fi

# Всегда возвращаем успех если тесты прошли
exit 0