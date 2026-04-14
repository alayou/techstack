package utils

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// IsFileInputValid returns true this is a valid file name.
// This method must be used before joining a file name, generally provided as
// user input, with a directory
func IsFileInputValid(fileInput string) bool {
	cleanInput := filepath.Clean(fileInput)
	if cleanInput == "." || cleanInput == ".." {
		return false
	}
	return true
}

// ReadConfigFromFile reads a configuration parameter from the specified file
func ReadConfigFromFile(name, configDir string) (string, error) {
	if !IsFileInputValid(name) {
		return "", fmt.Errorf("invalid file input: %q", name)
	}
	if configDir == "" {
		if !filepath.IsAbs(name) {
			return "", fmt.Errorf("%q must be an absolute file path", name)
		}
	} else {
		if name != "" && !filepath.IsAbs(name) {
			name = filepath.Join(configDir, name)
		}
	}
	val, err := os.ReadFile(filepath.Clean(name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(val)), nil
}

func EmbedFS2Files(efs embed.FS) map[string][]byte {
	files := make(map[string][]byte)

	fs.WalkDir(efs, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			var body []byte
			body, err = efs.ReadFile(p)
			if err != nil {
				return err
			}
			files[p] = body
		}
		return nil
	})
	return files
}
