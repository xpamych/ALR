// ALR - Any Linux Repository
// Copyright (C) 2025 The ALR Authors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package depver

import "fmt"

// ForManager formats the dependency for a specific package manager.
// Different package managers have different syntax for version constraints:
//
//	pacman (Arch):     "gcc>=5.0" (no changes)
//	apt (Debian):      "gcc" (version ignored for install command)
//	dnf/yum (Fedora):  "gcc >= 5.0" (with spaces)
//	apk (Alpine):      "gcc>=5.0" (no changes)
//	zypper (openSUSE): "gcc >= 5.0" (with spaces)
//	apt-rpm (ALT):     "gcc >= 5.0" (with spaces)
func (d Dependency) ForManager(managerName string) string {
	if d.Name == "" {
		return ""
	}

	// No version constraint - just return the name
	if d.Operator == OpNone || d.Version == "" {
		return d.Name
	}

	switch managerName {
	case "apt":
		// APT doesn't support version constraints in 'apt install' command
		// Versions are checked after installation
		return d.Name

	case "pacman":
		// Pacman uses PKGBUILD-style: package>=version (no spaces)
		return fmt.Sprintf("%s%s%s", d.Name, d.Operator, d.Version)

	case "apk":
		// Alpine APK uses similar syntax to pacman
		return fmt.Sprintf("%s%s%s", d.Name, d.Operator, d.Version)

	case "dnf", "yum":
		// DNF/YUM use RPM-style: "package >= version" (with spaces)
		return fmt.Sprintf("%s %s %s", d.Name, d.Operator, d.Version)

	case "zypper":
		// Zypper uses RPM-style with spaces
		return fmt.Sprintf("%s %s %s", d.Name, d.Operator, d.Version)

	case "apt-rpm":
		// ALT Linux apt-rpm uses RPM-style
		return fmt.Sprintf("%s %s %s", d.Name, d.Operator, d.Version)

	default:
		// Default: PKGBUILD-style (no spaces)
		return fmt.Sprintf("%s%s%s", d.Name, d.Operator, d.Version)
	}
}

// ForManagerMultiple formats multiple dependencies for a specific package manager.
func ForManagerMultiple(deps []Dependency, managerName string) []string {
	result := make([]string, 0, len(deps))
	for _, dep := range deps {
		if formatted := dep.ForManager(managerName); formatted != "" {
			result = append(result, formatted)
		}
	}
	return result
}

// ForNfpm formats the dependency for nfpm package building.
// Different package formats have different dependency syntax:
//
//	deb:       "package (>= version)"
//	rpm:       "package >= version"
//	apk:       "package>=version"
//	archlinux: "package>=version"
func (d Dependency) ForNfpm(pkgFormat string) string {
	if d.Name == "" {
		return ""
	}

	// No version constraint - just return the name
	if d.Operator == OpNone || d.Version == "" {
		return d.Name
	}

	switch pkgFormat {
	case "deb":
		// Debian uses: package (>= version)
		return fmt.Sprintf("%s (%s %s)", d.Name, d.Operator, d.Version)

	case "rpm":
		// RPM uses: package >= version
		return fmt.Sprintf("%s %s %s", d.Name, d.Operator, d.Version)

	case "apk":
		// Alpine uses: package>=version
		return fmt.Sprintf("%s%s%s", d.Name, d.Operator, d.Version)

	case "archlinux":
		// Arch uses: package>=version
		return fmt.Sprintf("%s%s%s", d.Name, d.Operator, d.Version)

	default:
		// Default: no spaces
		return fmt.Sprintf("%s%s%s", d.Name, d.Operator, d.Version)
	}
}

// ForNfpmMultiple formats multiple dependencies for nfpm.
func ForNfpmMultiple(deps []Dependency, pkgFormat string) []string {
	result := make([]string, 0, len(deps))
	for _, dep := range deps {
		if formatted := dep.ForNfpm(pkgFormat); formatted != "" {
			result = append(result, formatted)
		}
	}
	return result
}
