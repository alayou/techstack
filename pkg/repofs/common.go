package repofs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// copyDir复制 afero.IOFS 目录到目标路径
func copyDir(srcFs afero.IOFS, srcPath, destPath string) error {
	// fs.WalkDir requires root to be "." for fs.FS implementations like afero.IOFS
	// Normalize "/" to "." to walk from the FS root
	walkRoot := srcPath
	relBase := srcPath
	if walkRoot == "/" {
		walkRoot = "."
		relBase = "."
	}

	return fs.WalkDir(srcFs, walkRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(relBase, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		destFile := filepath.Join(destPath, relPath)

		if d.IsDir() {
			fmt.Println("createDir", destFile)

			err := os.MkdirAll(destFile, 0755)
			if err != nil {
				return err
			}
			// 目录需要写权限才能在其中创建文件
			return nil
		}

		// For afero.IOFS, paths must be relative without leading slash
		openPath := path
		if len(openPath) > 0 && openPath[0] == '/' {
			openPath = openPath[1:]
		}
		if openPath == "" {
			openPath = "."
		}

		srcFile, err := srcFs.Open(openPath)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		fmt.Println("Create", destFile)

		destFileHandle, err := os.Create(destFile)
		if err != nil {
			return err
		}
		defer destFileHandle.Close()

		_, err = io.Copy(destFileHandle, srcFile)
		return err
	})
}
