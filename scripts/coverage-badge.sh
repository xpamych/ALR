# ALR - Any Linux Repository
# Copyright (C) 2025 Евгений Храмов
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


#!/bin/bash
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

COLOR="#4c1"
if (( $(echo "$COVERAGE < 50" | bc -l) )); then
    COLOR="#e05d44"
elif (( $(echo "$COVERAGE < 80" | bc -l) )); then
    COLOR="#dfb317"
fi

cat <<EOF > assets/coverage-badge.svg
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="109" height="20">
    <linearGradient id="smooth" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/></linearGradient>
    <mask id="round">
        <rect width="109" height="20" rx="3" fill="#fff"/>
    </mask>
    <g mask="url(#round)"><rect width="65" height="20" fill="#555"/>
        <rect x="65" width="44" height="20" fill="${COLOR}"/>
        <rect width="109" height="20" fill="url(#smooth)"/>
    </g>
    <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
        <text x="33.5" y="15" fill="#010101" fill-opacity=".3">coverage</text>
        <text x="33.5" y="14">coverage</text>
        <text x="86" y="15" fill="#010101" fill-opacity=".3">${COVERAGE}%</text>
        <text x="86" y="14">${COVERAGE}%</text>
    </g>
</svg>
EOF