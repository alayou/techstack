package gitnexusctl

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/alayou/techstack/model"
	"github.com/spf13/afero"
)

func TestHasGitNexus(t *testing.T) {
	if !HasGitNexus() {
		t.Error("Not Found gitNexus")
	}
}
func getLlmModelConfig() (cfg model.LLMModelConfig, err error) {
	err = json.Unmarshal([]byte(`{
  "apiKey": "",
  "baseUrl": "https://api.minimaxi.com/v1",
  "provider": "openai",
  "model": "MiniMax-M2.7"
}`), &cfg)
	return
}
func TestWiki(t *testing.T) {
	dir, _ := os.Getwd()
	cfg, _ := getLlmModelConfig()
	out, err := Wiki(afero.NewBasePathFs(afero.NewOsFs(), dir), &cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(out)
}
