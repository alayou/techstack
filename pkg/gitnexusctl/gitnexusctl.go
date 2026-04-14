package gitnexusctl

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/alayou/techstack/model"
	"github.com/spf13/afero"
)

// hasGitNexus 检查是否安装 gitnexus
func HasGitNexus() bool {
	_, err := exec.LookPath("gitnexus")
	return err == nil
}

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
		destFileHandle, err := os.Create(destFile)
		if err != nil {
			return err
		}
		defer destFileHandle.Close()

		_, err = io.Copy(destFileHandle, srcFile)
		return err
	})
}

// readWikiFiles 读取 wiki 目录下所有 markdown 文件并合并
func readWikiFiles(wikiPath string) (string, error) {
	var result string

	err := filepath.Walk(wikiPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".md" && ext != ".markdown" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(wikiPath, path)
		result += fmt.Sprintf("## %s\n\n", relPath)
		result += string(content) + "\n\n"

		return nil
	})

	return result, err
}

// Wiki 生成文本 ，由于gitnexus 必须使用git仓库，这个方案可以用于本地分析
func Wiki(repoFs afero.Fs, llmConfig *model.LLMModelConfig) (string, error) {
	// 1. gitnexus 无法直接读取 afero.Fs
	// 解决方法： 将repoFs 输出到tmp目录，然后gitnexus去读取

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "gitnexus-wiki-*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 将 repoFs 复制到临时目录
	if err := copyDir(afero.NewIOFS(repoFs), "/", tmpDir); err != nil {
		return "", fmt.Errorf("复制仓库到临时目录失败: %v", err)
	}
	// 2. 调用 gitnexus 生成文档
	cmd := exec.Command("gitnexus", "wiki",
		"--provider", llmConfig.Provider,
		"--model", llmConfig.Model,
		"--api-key", llmConfig.ApiKey,
		"--base-url", llmConfig.BaseUrl,
		tmpDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gitnexus 执行失败: %v, %s", err, string(output))
	}

	// 3. gitnexus 文档直接输出到 os 文件系统中
	// 读取 tmp 目录对应的 .gitnexus/wiki 文件合并后输出
	wikiPath := filepath.Join(tmpDir, ".gitnexus", "wiki")
	if _, err := os.Stat(wikiPath); os.IsNotExist(err) {
		return "", fmt.Errorf("wiki 目录不存在: %s", wikiPath)
	}

	wikiContent, err := readWikiFiles(wikiPath)
	if err != nil {
		return "", fmt.Errorf("读取 wiki 文件失败: %v", err)
	}

	return wikiContent, nil
}
