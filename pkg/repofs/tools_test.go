package repofs

import (
	"os"
	"testing"

	"github.com/spf13/afero"
)

func TestNewLLMFsTools(t *testing.T) {
	dir, _ := os.Getwd()
	tools := NewLLMFsTools(afero.NewBasePathFs(afero.NewOsFs(), dir))
	for _, tool := range tools {
		if tool.Name() == "list_files" {
			ls, err := tool.Call(t.Context(), "")
			if err != nil {
				t.Fatal(err)
			}
			t.Log(ls)
		}
		if tool.Name() == "read_file" {
			res, _ := tool.Call(t.Context(), "tools_test.go")
			t.Log(res)
		}
	}
}
