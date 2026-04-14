package bind

// ==============================
// Package（全局库）请求结构体
// ==============================
type CreatePackageRequest struct {
	Name        string `json:"name" validate:"required"`
	PurlType    string `json:"purl_type" validate:"required"` // npm/golang/pypi/cargo
	Version     string `json:"version"`
	Description string `json:"description"`
	HomepageURL string `json:"homepage_url"`
	RepoURL     string `json:"repo_url"`
}
type BatchImportPackageRequest struct {
	List []CreatePackageRequest `json:"list"` // 最大 5000
}
