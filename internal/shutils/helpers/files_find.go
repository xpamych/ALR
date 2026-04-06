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

package helpers

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func matchNamePattern(name, pattern string) bool {
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		return false
	}
	return matched
}

// escapeGlob escapes special glob characters (*, ?, [, ], {, }) in a path
// so that they are treated as literal characters instead of glob patterns.
// This is needed when a directory or filename contains these characters.
func escapeGlob(path string) string {
	var buf strings.Builder
	for _, r := range path {
		switch r {
		case '*', '?', '[', ']', '{', '}':
			buf.WriteRune('\\')
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

// splitGlobPath splits a path into a base directory and a glob pattern.
// It finds the first glob special character (* or ?) and splits there.
// Square brackets [ ] and curly braces { } are always treated as literal
// characters since they appear in real file paths (e.g., "config[amd64].txt",
// "libfoo.so.{version}").
func splitGlobPath(searchPath string) (basepath, pattern string) {
	// Find the first * or ? which indicates the start of a glob pattern
	for i, r := range searchPath {
		if r == '*' || r == '?' {
			// Find the last separator before this position
			lastSep := strings.LastIndexAny(searchPath[:i], `/\`)
			if lastSep == -1 {
				return ".", searchPath
			}
			basepath = searchPath[:lastSep+1] // include the separator
			pattern = searchPath[lastSep+1:]
			return basepath, pattern
		}
	}
	// No glob characters found - the entire path is the base
	return searchPath, ""
}

func validateDir(dirPath, commandName string) error {
	info, err := os.Stat(dirPath)
	if err != nil {
		return fmt.Errorf("%s: %w", commandName, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s: %s is not a directory", commandName, dirPath)
	}
	return nil
}

func outputFiles(hc interp.HandlerContext, files []string) error {
	for _, file := range files {
		v, err := syntax.Quote(file, syntax.LangAuto)
		if err != nil {
			return err
		}
		fmt.Fprintln(hc.Stdout, v)
	}
	return nil
}

func makeRelativePath(basePath, fullPath string) (string, error) {
	relPath, err := filepath.Rel(basePath, fullPath)
	if err != nil {
		return "", err
	}
	return "./" + relPath, nil
}

func filesFindLangCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*.mo"
	if len(args) > 0 {
		namePattern = args[0] + ".mo"
	}

	localePath := "./usr/share/locale/"
	realPath := path.Join(hc.Dir, localePath)

	if err := validateDir(realPath, "files-find-lang"); err != nil {
		return err
	}

	var langFiles []string
	err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
			relPath, relErr := makeRelativePath(hc.Dir, p)
			if relErr != nil {
				return relErr
			}
			langFiles = append(langFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("files-find-lang: %w", err)
	}

	return outputFiles(hc, langFiles)
}

func filesFindDocCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	if len(args) > 0 {
		namePattern = args[0]
	}

	docPath := "./usr/share/doc/"
	docRealPath := path.Join(hc.Dir, docPath)

	if err := validateDir(docRealPath, "files-find-doc"); err != nil {
		return err
	}

	var docFiles []string

	entries, err := os.ReadDir(docRealPath)
	if err != nil {
		return fmt.Errorf("files-find-doc: %w", err)
	}

	for _, entry := range entries {
		if matchNamePattern(entry.Name(), namePattern) {
			targetPath := filepath.Join(docRealPath, entry.Name())
			targetInfo, err := os.Stat(targetPath)
			if err != nil {
				return fmt.Errorf("files-find-doc: %w", err)
			}
			if targetInfo.IsDir() {
				err := filepath.Walk(targetPath, func(subPath string, subInfo os.FileInfo, subErr error) error {
					if subErr != nil {
						return subErr
					}
					relPath, err := makeRelativePath(hc.Dir, subPath)
					if err != nil {
						return err
					}
					docFiles = append(docFiles, relPath)
					return nil
				})
				if err != nil {
					return fmt.Errorf("files-find-doc: %w", err)
				}
			}
		}
	}

	return outputFiles(hc, docFiles)
}

func filesFindCmd(hc interp.HandlerContext, cmd string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("files-find: at least one glob pattern is required")
	}

	var foundFiles []string

	for _, globPattern := range args {
		searchPath := path.Join(hc.Dir, globPattern)

		// Split the path into base directory and glob pattern.
		// We treat [ and ] as literal characters since they appear in real file paths.
		basepath, pattern := splitGlobPath(searchPath)

		// Check if basepath exists (use Lstat to not follow symlinks)
		info, err := os.Lstat(basepath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("files-find: %w", doublestar.ErrPatternNotExist)
			}
			return fmt.Errorf("files-find: %w", err)
		}
		// If pattern is not empty, basepath must be a directory
		if pattern != "" && !info.IsDir() {
			return fmt.Errorf("files-find: %s is not a directory", basepath)
		}

		// If pattern is empty, we're looking for a specific file/directory
		if pattern == "" {
			relPath, err := makeRelativePath(hc.Dir, basepath)
			if err != nil {
				return err
			}
			foundFiles = append(foundFiles, relPath)
			continue
		}

		// Use filepath.Walk to traverse the directory and match files manually.
		// This approach correctly handles files and directories with glob special
		// characters ([, ], {, }, *, ?) in their names, treating them as literals.
		err = filepath.Walk(basepath, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip the base directory itself
			if p == basepath {
				return nil
			}

			// Get the relative path from basepath for pattern matching
			relFromBase, err := filepath.Rel(basepath, p)
			if err != nil {
				return err
			}

			// Escape glob special characters in the file path so they are treated
			// as literal characters when matching against the pattern.
			// This fixes matching when filenames or directory names contain [, ], *, ?
			escapedRelPath := escapeGlob(relFromBase)

			// Match the escaped relative path against the pattern
			matched, err := doublestar.Match(pattern, escapedRelPath)
			if err != nil {
				return fmt.Errorf("files-find: pattern error: %w", err)
			}

			if matched {
				relPath, err := makeRelativePath(hc.Dir, p)
				if err != nil {
					return err
				}
				foundFiles = append(foundFiles, relPath)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("files-find: %w", err)
		}
	}

	return outputFiles(hc, foundFiles)
}

func filesFindBinCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	if len(args) > 0 {
		namePattern = args[0]
	}

	binPath := "./usr/bin/"
	realPath := path.Join(hc.Dir, binPath)

	if err := validateDir(realPath, "files-find-bin"); err != nil {
		return err
	}

	var binFiles []string
	err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
			relPath, relErr := makeRelativePath(hc.Dir, p)
			if relErr != nil {
				return relErr
			}
			binFiles = append(binFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("files-find-bin: %w", err)
	}

	return outputFiles(hc, binFiles)
}

func filesFindLibCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	if len(args) > 0 {
		namePattern = args[0]
	}

	libPaths := []string{"./lib/", "./lib64/", "./usr/lib/", "./usr/lib64/", "./usr/local/lib/", "./usr/local/lib64/"}
	var libFiles []string

	for _, libPath := range libPaths {
		realPath := path.Join(hc.Dir, libPath)
		if _, err := os.Stat(realPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
				relPath, relErr := makeRelativePath(hc.Dir, p)
				if relErr != nil {
					return relErr
				}
				libFiles = append(libFiles, relPath)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("files-find-lib: %w", err)
		}
	}

	return outputFiles(hc, libFiles)
}

func filesFindIncludeCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	if len(args) > 0 {
		namePattern = args[0]
	}

	includePath := "./usr/include/"
	realPath := path.Join(hc.Dir, includePath)

	if err := validateDir(realPath, "files-find-include"); err != nil {
		return err
	}

	var includeFiles []string
	err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
			relPath, relErr := makeRelativePath(hc.Dir, p)
			if relErr != nil {
				return relErr
			}
			includeFiles = append(includeFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("files-find-include: %w", err)
	}

	return outputFiles(hc, includeFiles)
}

func filesFindShareCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	sharePath := "./usr/share/"

	if len(args) > 0 {
		if len(args) == 1 {
			sharePath = "./usr/share/" + args[0] + "/"
		} else {
			sharePath = "./usr/share/" + args[0] + "/"
			namePattern = args[1]
		}
	}

	realPath := path.Join(hc.Dir, sharePath)

	if err := validateDir(realPath, "files-find-share"); err != nil {
		return err
	}

	var shareFiles []string
	err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
			relPath, relErr := makeRelativePath(hc.Dir, p)
			if relErr != nil {
				return relErr
			}
			shareFiles = append(shareFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("files-find-share: %w", err)
	}

	return outputFiles(hc, shareFiles)
}

func filesFindManCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	manSection := "*"

	if len(args) > 0 {
		if len(args) == 1 {
			manSection = args[0]
		} else {
			manSection = args[0]
			namePattern = args[1]
		}
	}

	manPath := "./usr/share/man/man" + manSection + "/"
	realPath := path.Join(hc.Dir, manPath)

	if err := validateDir(realPath, "files-find-man"); err != nil {
		return err
	}

	var manFiles []string
	err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
			relPath, relErr := makeRelativePath(hc.Dir, p)
			if relErr != nil {
				return relErr
			}
			manFiles = append(manFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("files-find-man: %w", err)
	}

	return outputFiles(hc, manFiles)
}

func filesFindConfigCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	if len(args) > 0 {
		namePattern = args[0]
	}

	configPath := "./etc/"
	realPath := path.Join(hc.Dir, configPath)

	if err := validateDir(realPath, "files-find-config"); err != nil {
		return err
	}

	var configFiles []string
	err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
			relPath, relErr := makeRelativePath(hc.Dir, p)
			if relErr != nil {
				return relErr
			}
			configFiles = append(configFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("files-find-config: %w", err)
	}

	return outputFiles(hc, configFiles)
}

func filesFindSystemdCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	if len(args) > 0 {
		namePattern = args[0]
	}

	systemdPath := "./usr/lib/systemd/system/"
	realPath := path.Join(hc.Dir, systemdPath)

	if err := validateDir(realPath, "files-find-systemd"); err != nil {
		return err
	}

	var systemdFiles []string
	err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
			relPath, relErr := makeRelativePath(hc.Dir, p)
			if relErr != nil {
				return relErr
			}
			systemdFiles = append(systemdFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("files-find-systemd: %w", err)
	}

	return outputFiles(hc, systemdFiles)
}

func filesFindSystemdUserCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	if len(args) > 0 {
		namePattern = args[0]
	}

	systemdUserPath := "./usr/lib/systemd/user/"
	realPath := path.Join(hc.Dir, systemdUserPath)

	if err := validateDir(realPath, "files-find-systemd-user"); err != nil {
		return err
	}

	var systemdUserFiles []string
	err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
			relPath, relErr := makeRelativePath(hc.Dir, p)
			if relErr != nil {
				return relErr
			}
			systemdUserFiles = append(systemdUserFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("files-find-systemd-user: %w", err)
	}

	return outputFiles(hc, systemdUserFiles)
}

func filesFindLicenseCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	if len(args) > 0 {
		namePattern = args[0]
	}

	licensePath := "./usr/share/licenses/"
	realPath := path.Join(hc.Dir, licensePath)

	if err := validateDir(realPath, "files-find-license"); err != nil {
		return err
	}

	var licenseFiles []string
	err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
			relPath, relErr := makeRelativePath(hc.Dir, p)
			if relErr != nil {
				return relErr
			}
			licenseFiles = append(licenseFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("files-find-license: %w", err)
	}

	return outputFiles(hc, licenseFiles)
}

func filesFindSbinCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	if len(args) > 0 {
		namePattern = args[0]
	}

	sbinPaths := []string{"./sbin/", "./usr/sbin/"}
	var sbinFiles []string

	for _, sbinPath := range sbinPaths {
		realPath := path.Join(hc.Dir, sbinPath)
		if _, err := os.Stat(realPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
				relPath, relErr := makeRelativePath(hc.Dir, p)
				if relErr != nil {
					return relErr
				}
				sbinFiles = append(sbinFiles, relPath)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("files-find-sbin: %w", err)
		}
	}

	return outputFiles(hc, sbinFiles)
}

func filesFindIconsCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	if len(args) > 0 {
		namePattern = args[0]
	}

	iconsPath := "./usr/share/icons/"
	realPath := path.Join(hc.Dir, iconsPath)

	if err := validateDir(realPath, "files-find-icons"); err != nil {
		return err
	}

	var iconsFiles []string
	err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
			relPath, relErr := makeRelativePath(hc.Dir, p)
			if relErr != nil {
				return relErr
			}
			iconsFiles = append(iconsFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("files-find-icons: %w", err)
	}

	return outputFiles(hc, iconsFiles)
}

func filesFindDesktopCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	if len(args) > 0 {
		namePattern = args[0]
	}

	desktopPath := "./usr/share/applications/"
	realPath := path.Join(hc.Dir, desktopPath)

	if err := validateDir(realPath, "files-find-desktop"); err != nil {
		return err
	}

	var desktopFiles []string
	err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
			relPath, relErr := makeRelativePath(hc.Dir, p)
			if relErr != nil {
				return relErr
			}
			desktopFiles = append(desktopFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("files-find-desktop: %w", err)
	}

	return outputFiles(hc, desktopFiles)
}

func filesFindDbusCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	if len(args) > 0 {
		namePattern = args[0]
	}

	dbusPath := "./usr/share/dbus-1/"
	realPath := path.Join(hc.Dir, dbusPath)

	if err := validateDir(realPath, "files-find-dbus"); err != nil {
		return err
	}

	var dbusFiles []string
	err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
			relPath, relErr := makeRelativePath(hc.Dir, p)
			if relErr != nil {
				return relErr
			}
			dbusFiles = append(dbusFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("files-find-dbus: %w", err)
	}

	return outputFiles(hc, dbusFiles)
}

func filesFindPolkitCmd(hc interp.HandlerContext, cmd string, args []string) error {
	namePattern := "*"
	if len(args) > 0 {
		namePattern = args[0]
	}

	polkitPath := "./usr/share/polkit-1/"
	realPath := path.Join(hc.Dir, polkitPath)

	if err := validateDir(realPath, "files-find-polkit"); err != nil {
		return err
	}

	var polkitFiles []string
	err := filepath.Walk(realPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && matchNamePattern(info.Name(), namePattern) {
			relPath, relErr := makeRelativePath(hc.Dir, p)
			if relErr != nil {
				return relErr
			}
			polkitFiles = append(polkitFiles, relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("files-find-polkit: %w", err)
	}

	return outputFiles(hc, polkitFiles)
}
