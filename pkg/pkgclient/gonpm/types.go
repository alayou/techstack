package gonpm

// DistTags 表示npm包的发行标签信息
type DistTags struct {
	Latest string `json:"latest"` // 最新版本号
}

// Bugs 表示包的bug报告信息
type Bugs struct {
	Url string `json:"url"` // bug报告链接
}

// Author 表示包的作者信息
type Author struct {
	Name  string `json:"name"`  // 作者名称
	Email string `json:"email"` // 作者邮箱
}

// Repository 表示包的代码仓库信息
type Repository struct {
	Url  string `json:"url"`  // 仓库地址
	Type string `json:"type"` // 仓库类型（如git）
}

// Package 表示npm包的完整元数据信息
type Package struct {
	Id             string             `json:"_id"`            // 包唯一标识
	Rev            string             `json:"_rev"`           // 包修订版本
	Name           string             `json:"name"`           // 包名称
	DistTags       DistTags           `json:"dist-tags"`      // 发行标签
	Bugs           Bugs               `json:"bugs"`           // bug信息
	Author         Author             `json:"author"`         // 作者
	License        string             `json:"license"`        // 许可证
	Homepage       string             `json:"homepage"`       // 主页
	Keywords       []string           `json:"keywords"`       // 关键词
	Repository     Repository         `json:"repository"`     // 仓库信息
	Description    string             `json:"description"`    // 包描述
	Contributors   []Author           `json:"contributors"`   // 贡献者列表
	Maintainers    []Author           `json:"maintainers"`    // 维护者列表
	Readme         string             `json:"readme"`         // README内容
	ReadmeFilename string             `json:"readmeFilename"` // README文件名
	Dist           Dist               `json:"dist"`           // 分发信息
	Versions       map[string]Version `json:"versions"`       // 版本列表
	Time           map[string]string  `json:"-"`
	Users          map[string]bool    `json:"-"`
}

// SignaturesItem 表示包的签名信息（用于安全验证）
type SignaturesItem struct {
	Sig   string `json:"sig"`   // 签名内容
	Keyid string `json:"keyid"` // 公钥标识
}

// Dist 表示包的分发信息
type Dist struct {
	Shasum       string           `json:"shasum"`       // SHA校验和
	Tarball      string           `json:"tarball"`      // tarball下载链接
	FileCount    int64            `json:"fileCount"`    // 文件数量
	Integrity    string           `json:"integrity"`    // 完整性哈希
	Signatures   []SignaturesItem `json:"signatures"`   // 签名列表
	UnpackedSize int64            `json:"unpackedSize"` // 解压后大小
}

// Version 表示包的单个版本信息
type Version struct {
	Name         string   `json:"name"`         // 包名
	Version      string   `json:"version"`      // 版本号
	Keywords     []string `json:"keywords"`     // 关键词
	Author       Author   `json:"author"`       // 作者
	License      string   `json:"license"`      // 许可证
	Id           string   `json:"_id"`          // 版本唯一标识
	Maintainers  []Author `json:"maintainers"`  // 维护者列表
	Contributors []Author `json:"contributors"` // 贡献者列表
	Homepage     string   `json:"homepage"`     // 主页
	Bugs         Bugs     `json:"bugs"`         // bug信息
	Dist         Dist     `json:"dist"`         // 分发信息
	Icon         string   `json:"icon"`         // 图标
	Main         string   `json:"main"`         // 入口文件
	GitHead      string   `json:"gitHead"`      // Git提交哈希
	Scripts      struct {
		Test string `json:"test"` // 测试脚本
	} `json:"scripts"`
	NpmUser                Author     `json:"_npmUser"`       // NPM用户名
	Repository             Repository `json:"repository"`     // 仓库信息
	NpmVersion             string     `json:"_npmVersion"`    // NPM版本
	Description            string     `json:"description"`    // 版本描述
	Directories            any        `json:"directories"`    // 目录结构
	NodeVersion            string     `json:"_nodeVersion"`   // 所需Node版本
	HasShrinkwrap          bool       `json:"_hasShrinkwrap"` // 是否有shrinkwrap
	NpmOperationalInternal struct {
		Tmp  string `json:"tmp"`  // 临时目录
		Host string `json:"host"` // 主机信息
	} `json:"_npmOperationalInternal"`
}
