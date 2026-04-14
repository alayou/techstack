//go:build !windows
// +build !windows

package service

import (
	"os"
	"path/filepath"

	"github.com/alayou/techstack/global"
)

// helper function returns the default configuration path
// for the drone configuration.
func configPath() string {
	if fileExist(global.Config.ConfigFile) {
		return global.Config.ConfigFile
	}
	if filepath.IsAbs(global.DefaultConfigPath) {
		return global.DefaultConfigPath
	}
	return filepath.Join("/opt", global.DefaultConfigPath)
}

func fileExist(file string) bool {
	_, err := os.Stat(file)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return true
}
