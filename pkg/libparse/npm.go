package libparse

import (
	"encoding/json"
)

var requiredFields = []string{"name", "version"}

type PackageJsonFile struct {
	Name                 string            `json:"name"`
	Version              string            `json:"version"`
	Private              bool              `json:"private"`
	Description          string            `json:"description"`
	Keywords             []string          `json:"keywords"`
	Homepage             string            `json:"homepage"`
	Author               any               `json:"author"`
	Contributors         any               `json:"contributors"`
	Funding              any               `json:"funding"`
	License              string            `json:"license"`
	Main                 string            `json:"main"`
	Bin                  any               `json:"bin"`
	Browser              any               `json:"browser"`
	Man                  string            `json:"man"`
	Directories          any               `json:"directories"`
	Files                []string          `json:"files"`
	Scripts              map[string]string `json:"scripts"`
	Os                   []string          `json:"os"`
	Cpu                  []string          `json:"cpu"`
	Repository           string            `json:"repository"`
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
	BundleDependencies   map[string]string `json:"bundleDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
}

// ParsePackageJson 解析 package.json 文件
func ParsePackageJson(content string) ([]PackageInfo, error) {
	var packages []PackageInfo

	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(content), &jsonData); err != nil {
		return nil, err
	}

	// 获取 dependencies 和 devDependencies
	for _, key := range []string{"dependencies", "devDependencies"} {
		if deps, ok := jsonData[key].(map[string]interface{}); ok {
			for name, version := range deps {
				verStr := ""
				if v, ok := version.(string); ok {
					verStr = v
				}
				packages = append(packages, PackageInfo{
					Ecosystem: "npm",
					Name:      name,
					Version:   verStr,
				})
			}
		}
	}

	return packages, nil
}
