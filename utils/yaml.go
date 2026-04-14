package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadYAMLConfig 加载yaml配置文件，如果没有则将默认配置写入文件.
func LoadYAMLConfig(name string, c interface{}) (string, error) {
	if filepath.IsAbs(name) {
		if !fileExist(name) {
			body, err := yaml.Marshal(c)
			if err != nil {
				return "", err
			}
			return "", os.WriteFile(name, body, 0600)
		}
	} else {
		location, err := os.Executable()
		if err != nil {
			return "", err
		}
		name = filepath.Join(filepath.Dir(location), name)
		if !fileExist(name) {
			location, err := os.Getwd()
			if err != nil {
				return "", err
			}
			_, name = filepath.Split(name)
			name = filepath.Join(location, name)
			if !fileExist(name) {
				return "", fmt.Errorf("配置文件%s不存在", name)
			}
		}
	}
	file, err := os.Open(filepath.Clean(name))
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	buffer := &bytes.Buffer{}
	_, err = io.Copy(io.MultiWriter(hash, buffer), file)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), yaml.NewDecoder(buffer).Decode(c)
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
