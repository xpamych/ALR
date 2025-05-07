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


VULNS_FILE="vulns.json"
COMMIT_MSG_FILE="commit_msg.txt"

echo "Scanning for vulnerabilities with Trivy..."
trivy fs --scanners vuln --format json . > "$VULNS_FILE"

echo "security: update vulnerable packages" > "$COMMIT_MSG_FILE"
echo "" >> "$COMMIT_MSG_FILE"
echo "Vulnerabilities detected by Trivy scan:" >> "$COMMIT_MSG_FILE"

echo "Processing vulnerabilities..."
jq -r '
  .Results[].Vulnerabilities[] | 
  select(.PkgName and .FixedVersion) | 
  "\(.PkgName)|\(.FixedVersion)|\(.VulnerabilityID)"
' "$VULNS_FILE" | sort | uniq | while IFS="|" read -r pkg version cve; do
  echo "- ${pkg} (${cve})" >> "$COMMIT_MSG_FILE"
  echo "Updating ${pkg} to v${version} (${cve})..."
  go get "${pkg}@v${version}" || echo "Failed to update ${pkg}"
done

echo "Running go mod tidy..."
go mod tidy

echo "Verifying fixes..."
trivy fs --scanners vuln .

echo ""
echo "Suggested commit message:"
echo "------------------------"
cat "$COMMIT_MSG_FILE"
echo "------------------------"

rm "$VULNS_FILE"

git add go.mod go.sum

echo ""
echo "To commit these changes, run:"
echo "git commit -a -F $(pwd)/$COMMIT_MSG_FILE"