package model

type BrowseFileItem struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	IsDir     bool   `json:"is_dir"`
	Size      int64  `json:"size"`
	UpdatedAt int64  `json:"updated_at"`
}

type BrowseFilesResponse struct {
	Path       string           `json:"path"`
	ParentPath *string          `json:"parent_path"`
	Search     string           `json:"search"`
	Limit      int              `json:"limit"`
	Truncated  bool             `json:"truncated"`
	Items      []BrowseFileItem `json:"items"`
}
