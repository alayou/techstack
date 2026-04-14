package gocargo

type Links struct {
	VersionDownloads    string `json:"version_downloads"`
	Versions            string `json:"versions"`
	Owners              string `json:"owners"`
	OwnerTeam           string `json:"owner_team"`
	OwnerUser           string `json:"owner_user"`
	ReverseDependencies string `json:"reverse_dependencies"`
}

type Crate struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updated_at"`

	CreatedAt      string    `json:"created_at"`
	Downloads      int64     `json:"downloads"`
	DefaultVersion string    `json:"default_version"`
	NumVersions    int64     `json:"num_versions"`
	Yanked         bool      `json:"yanked"`
	MaxVersion     string    `json:"max_version"`
	NewestVersion  string    `json:"newest_version"`
	Description    string    `json:"description"`
	Homepage       string    `json:"homepage"`
	Repository     string    `json:"repository"`
	Links          Links     `json:"links"`
	ExactMatch     bool      `json:"exact_match"`
	TrustpubOnly   bool      `json:"trustpub_only"`
	Versions       []Version `json:"-"`
	// Keywords         interface{}   `json:"keywords"`
	// Categories       interface{}   `json:"categories"`
	// Badges           []interface{} `json:"badges"`
	// Documentation    interface{}   `json:"documentation"`
	// RecentDownloads  interface{}   `json:"recent_downloads"`
	// MaxStableVersion interface{}   `json:"max_stable_version"`
}

func (s Crate) GetLicence() string {
	for _, v := range s.Versions {
		if v.License != "" {
			return v.License
		}
	}
	return ""
}
func (s Crate) GetHomepageURL() string {
	if s.Homepage != "" {
		return s.Homepage
	}
	for _, v := range s.Versions {
		homepage := v.GetHomepageURL()
		if homepage != "" {
			return homepage
		}
	}
	return s.Repository
}

type Features struct {
	Cargo            []string `json:"cargo"`
	Color            []string `json:"color"`
	Debug            []string `json:"debug"`
	Default          []string `json:"default"`
	Deprecated       []string `json:"deprecated"`
	Derive           []string `json:"derive"`
	Env              []string `json:"env"`
	ErrorContext     []string `json:"error-context"`
	Help             []string `json:"help"`
	Std              []string `json:"std"`
	String           []string `json:"string"`
	Suggestions      []string `json:"suggestions"`
	Unicode          []string `json:"unicode"`
	UnstableDoc      []string `json:"unstable-doc"`
	UnstableExt      []string `json:"unstable-ext"`
	UnstableMarkdown []string `json:"unstable-markdown"`
	UnstableStyles   []string `json:"unstable-styles"`
	UnstableV5       []string `json:"unstable-v5"`
	Usage            []string `json:"usage"`
	WrapHelp         []string `json:"wrap_help"`
}

type PublishedBy struct {
	Id     int64  `json:"id"`
	Login  string `json:"login"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Url    string `json:"url"`
}

type User struct {
	Id     int64  `json:"id"`
	Login  string `json:"login"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Url    string `json:"url"`
}

type Rust struct {
	CodeLines    int64 `json:"code_lines"`
	CommentLines int64 `json:"comment_lines"`
	Files        int64 `json:"files"`
}

type Languages struct {
	Rust Rust `json:"Rust"`
}

type Linecounts struct {
	Languages         Languages `json:"languages"`
	TotalCodeLines    int64     `json:"total_code_lines"`
	TotalCommentLines int64     `json:"total_comment_lines"`
}

type Version struct {
	Id    int64  `json:"id"`
	Crate string `json:"crate"`
	Num   string `json:"num"`
	// DlPath        string      `json:"dl_path"`
	ReadmePath    string      `json:"readme_path"`
	UpdatedAt     string      `json:"updated_at"`
	CreatedAt     string      `json:"created_at"`
	Downloads     int64       `json:"downloads"`
	Features      Features    `json:"features"`
	Yanked        bool        `json:"yanked"`
	License       string      `json:"license"`
	CrateSize     int64       `json:"crate_size"`
	PublishedBy   PublishedBy `json:"published_by"`
	Checksum      string      `json:"checksum"`
	RustVersion   string      `json:"rust_version"`
	HasLib        bool        `json:"has_lib"`
	BinNames      []string    `json:"bin_names"`
	Edition       string      `json:"edition"`
	Description   string      `json:"description"`
	Homepage      string      `json:"homepage"`
	Documentation string      `json:"documentation"`
	Repository    string      `json:"repository"`
	Linecounts    Linecounts  `json:"linecounts"`
}

func (v Version) GetHomepageURL() string {
	if v.Homepage != "" {
		return v.Homepage
	}
	if v.Documentation != "" {
		return v.Homepage
	}
	if v.Repository != "" {
		return v.Repository
	}
	return ""
}

type CrateDependency struct {
	Id              int64         `json:"id"`
	VersionId       int64         `json:"version_id"`
	CrateId         string        `json:"crate_id"`
	Req             string        `json:"req"`
	Optional        bool          `json:"optional"`
	DefaultFeatures bool          `json:"default_features"`
	Features        []interface{} `json:"features"`
	Target          interface{}   `json:"target"`
	Kind            string        `json:"kind"`
	Downloads       int64         `json:"downloads"`
}
