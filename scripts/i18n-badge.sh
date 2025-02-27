#!/bin/bash
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


TRANSLATIONS_DIR="internal/translations/po"

if [ ! -d "$TRANSLATIONS_DIR" ]; then
    echo "Error: directory '$TRANSLATIONS_DIR' not found"
    exit 1
fi

declare -A TOTAL_STRINGS_MAP
declare -A TRANSLATED_STRINGS_MAP

for PO_FILE in $(find "$TRANSLATIONS_DIR" -type f -name "*.po"); do
    LANG_DIR=$(dirname "$PO_FILE")
    LANG=$(basename "$LANG_DIR")

    STATS=$(LC_ALL=C msgfmt --statistics -o /dev/null "$PO_FILE" 2>&1)

    NUMBERS=($(echo "$STATS" | grep -o '[0-9]\+'))

    case ${#NUMBERS[@]} in
        1) TRANSLATED_STRINGS=${NUMBERS[0]}; UNTRANSLATED_STRINGS=0 ;;  # all translated
        2) TRANSLATED_STRINGS=${NUMBERS[0]}; UNTRANSLATED_STRINGS=${NUMBERS[1]} ;;  # no fuzzy
        3) TRANSLATED_STRINGS=${NUMBERS[0]}; UNTRANSLATED_STRINGS=${NUMBERS[2]} ;;  # with fuzzy
        *) TRANSLATED_STRINGS=0; UNTRANSLATED_STRINGS=0 ;; 
    esac

    TOTAL_STRINGS=$((TRANSLATED_STRINGS + UNTRANSLATED_STRINGS))

    TOTAL_STRINGS_MAP[$LANG]=$((TOTAL_STRINGS_MAP[$LANG] + TOTAL_STRINGS))
    TRANSLATED_STRINGS_MAP[$LANG]=$((TRANSLATED_STRINGS_MAP[$LANG] + TRANSLATED_STRINGS))
done

for LANG in "${!TOTAL_STRINGS_MAP[@]}"; do
    TOTAL=${TOTAL_STRINGS_MAP[$LANG]}
    TRANSLATED=${TRANSLATED_STRINGS_MAP[$LANG]}
    if [ "$TOTAL" -eq 0 ]; then
        PERCENTAGE="0.00"
    else
        PERCENTAGE=$(echo "scale=2; ($TRANSLATED / $TOTAL) * 100" | bc)
    fi
    COLOR="#4c1"
    if (( $(echo "$PERCENTAGE < 50" | bc -l) )); then
        COLOR="#e05d44"
    elif (( $(echo "$PERCENTAGE < 80" | bc -l) )); then
        COLOR="#dfb317"
    fi
cat <<EOF > assets/i18n-$LANG-badge.svg
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="129" height="20">
    <linearGradient id="smooth" x2="0" y2="100%"><stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/></linearGradient>
    <mask id="round">
        <rect width="129" height="20" rx="3" fill="#fff"/>
    </mask>
    <g mask="url(#round)">
        <rect width="75" height="20" fill="#555"/>
        <rect x="75" width="64" height="20" fill="${COLOR}"/>
        <rect width="129" height="20" fill="url(#smooth)"/>
    </g>
    <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
        <text x="37" y="15" fill="#010101" fill-opacity=".3">$LANG translate</text>
        <text x="37" y="14">$LANG translate</text>
        <text x="100" y="15" fill="#010101" fill-opacity=".3">${PERCENTAGE}%</text>
        <text x="100" y="14">${PERCENTAGE}%</text>
    </g>
</svg>
EOF
done
