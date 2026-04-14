package model

type Package struct {
	ID             ID     `json:"id,string,omitempty" gorm:"primaryKey;autoIncrement:false"` // 库ID
	Name           string `gorm:"size:255;not null;uniqueIndex:idx_purl" json:"name"`        // 包名
	PurlType       string `gorm:"size:50;not null;uniqueIndex:idx_purl" json:"purl_type"`    // 生态
	Description    string `gorm:"type:text" json:"description"`                              // 描述
	NormalizedName string `gorm:"type:text;not null" json:"normalized_name"`                 // NormalizedName 标准化后的包名称（统一小写）
	HomepageURL    string `gorm:"size:512" json:"homepage_url"`                              // 官网
	RepositoryURL  string `gorm:"size:512" json:"repository_url"`                            // 源码地址
	CreatedAt      int64  `json:"created_at" gorm:"not null;default:0"`                      // 创建时间
	UpdatedAt      int64  `json:"updated_at" gorm:"not null;default:0"`                      // 更新时间
}

func (Package) TableName() string {
	return PackageTableName
}

// PackageVersion
// 库版本
type PackageVersion struct {
	ID             ID      `json:"id,string,omitempty" gorm:"primaryKey;autoIncrement:false"`   // 版本ID
	PackageID      int64   `json:"package_id,string,omitempty" gorm:"index"`                    // 所属库
	Version        string  `json:"version" gorm:"size:100;not null"`                            // 版本号
	PURL           string  `json:"purl" gorm:"column:purl;size:512;not null;unique"`            // 完整PURL
	PublishedAt    int64   `json:"published_at"`                                                // 发布时间
	PublishedAtStr string  `json:"published_at_str"  gorm:"column:published_at_str;default:''"` // 发布时间
	Package        Package `gorm:"-" json:"-"`
}

func (PackageVersion) TableName() string {
	return PackageVersionTableName
}
