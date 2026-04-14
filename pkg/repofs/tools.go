package repofs

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"strings"

	"github.com/spf13/afero"
	"github.com/tmc/langchaingo/tools"
)

// ==============================================
// ReadFile Tool - 读取文件
// ==============================================

type fsReadFileFunc struct {
	memfs *afero.MemMapFs
}

func (s *fsReadFileFunc) Call(ctx context.Context, input string) (string, error) {
	// 尝试解析 JSON 输入（支持新的 schema）
	var filePath string
	params := make(map[string]any)
	if err := json.Unmarshal([]byte(input), &params); err == nil {
		if path, ok := params["path"].(string); ok {
			filePath = path
		}
	} else {
		// 兼容旧模式 - 直接使用字符串作为路径
		filePath = input
	}

	if filePath == "" {
		return "error: path parameter is required", nil
	}

	fd, err := s.memfs.Open(filePath)
	if err != nil {
		return "error: " + err.Error(), nil
	}
	defer fd.Close()
	sb := &strings.Builder{}
	_, err = io.Copy(sb, fd)
	if err != nil {
		return "error: " + err.Error(), nil
	}
	return sb.String(), err
}

func (s *fsReadFileFunc) Description() string {
	return "Read the full content of a file in the repository. " +
		"Use the 'path' parameter to specify the file path."
}

func (s *fsReadFileFunc) Name() string {
	return "read_file"
}

// GetParameters 实现 ToolWithSchema 接口
func (s *fsReadFileFunc) GetParameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path of the file to read",
			},
		},
		"required": []string{"path"},
	}
}

var _ tools.Tool = (*fsReadFileFunc)(nil)
var _ ToolWithSchema = (*fsReadFileFunc)(nil)

// ==============================================
// ListFiles Tool - 列出文件
// ==============================================

type fsListFilesFunc struct {
	memfs *afero.MemMapFs
}

func (s *fsListFilesFunc) Call(ctx context.Context, input string) (string, error) {
	files := s.memfs.ListNames()
	ls := make([]string, 0, len(files))
	// 忽略.git 文件
	for _, name := range files {
		if !strings.HasPrefix(name, ".git") {
			ls = append(ls, name)
		}
	}
	return strings.Join(ls, "\n"), nil
}

func (s *fsListFilesFunc) Description() string {
	return "List all files in the repository. " +
		"No parameters required. Returns all file paths separated by newlines."
}

func (s *fsListFilesFunc) Name() string {
	return "list_files"
}

// GetParameters 实现 ToolWithSchema 接口
func (s *fsListFilesFunc) GetParameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
		"required":   []string{},
	}
}

var _ tools.Tool = (*fsListFilesFunc)(nil)
var _ ToolWithSchema = (*fsListFilesFunc)(nil)

// ==============================================
// Afero FS ReadFile Tool
// ==============================================

type afsReadFileFunc struct {
	fs afero.IOFS
}

func (s *afsReadFileFunc) Call(ctx context.Context, input string) (string, error) {
	// 尝试解析 JSON 输入
	var filePath string
	params := make(map[string]any)
	if err := json.Unmarshal([]byte(input), &params); err == nil {
		if path, ok := params["path"].(string); ok {
			filePath = path
		}
	} else {
		// 兼容旧模式
		filePath = input
	}

	if filePath == "" {
		return "error: path parameter is required", nil
	}

	raw, err := s.fs.ReadFile(filePath)
	if err != nil {
		return "error: " + err.Error(), nil
	}
	return string(raw), err
}

func (s *afsReadFileFunc) Description() string {
	return "Read the full content of a file in the repository. " +
		"Use the 'path' parameter to specify the file path."
}

func (s *afsReadFileFunc) Name() string {
	return "read_file"
}

// GetParameters 实现 ToolWithSchema 接口
func (s *afsReadFileFunc) GetParameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path of the file to read",
			},
		},
		"required": []string{"path"},
	}
}

var _ tools.Tool = (*afsReadFileFunc)(nil)
var _ ToolWithSchema = (*afsReadFileFunc)(nil)

// ==============================================
// Afero FS ListFiles Tool
// ==============================================

type afsListFilesFunc struct {
	fs afero.IOFS
}

func (s *afsListFilesFunc) Call(ctx context.Context, input string) (string, error) {
	var files []string
	ignoreFiles := []string{".git", "/.git", ".cache", "/."}
	err := fs.WalkDir(s.fs, "/", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "/" {
			return nil
		}
		for _, file := range ignoreFiles {
			if strings.HasPrefix(path, file) {
				return nil
			}
		}
		if strings.Contains(path, "node_modules") {
			return nil
		}
		files = append(files, strings.TrimPrefix(path, "/"))
		return nil
	})
	if err != nil {
		return "error: " + err.Error(), nil
	}
	return strings.Join(files, "\n"), nil
}

func (s *afsListFilesFunc) Description() string {
	return "List all files in the repository. " +
		"No parameters required. Returns all file paths separated by newlines."
}

func (s *afsListFilesFunc) Name() string {
	return "list_files"
}

// GetParameters 实现 ToolWithSchema 接口
func (s *afsListFilesFunc) GetParameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
		"required":   []string{},
	}
}

var _ tools.Tool = (*afsListFilesFunc)(nil)
var _ ToolWithSchema = (*afsListFilesFunc)(nil)

// ==============================================
// Tool Constructor
// ==============================================

func NewLLMFsTools(afs afero.Fs) []tools.Tool {
	memfs, ok := afs.(*afero.MemMapFs)
	if ok {
		return []tools.Tool{&fsReadFileFunc{memfs: memfs}, &fsListFilesFunc{memfs: memfs}}
	}
	return []tools.Tool{&afsReadFileFunc{fs: afero.NewIOFS(afs)}, &afsListFilesFunc{fs: afero.NewIOFS(afs)}}
}

// ToolWithSchema 本地定义的接口，避免导入循环
type ToolWithSchema interface {
	tools.Tool
	GetParameters() map[string]any
}
