//go:build windows
// +build windows

package service

import (
	"os"

	"github.com/alayou/techstack/global"
)

// helper function returns the default configuration path
// for the drone configuration.
func configPath() string {
	if fileExist(global.Config.ConfigFile) {
		return global.Config.ConfigFile
	}
	return global.DefaultConfigPath
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
