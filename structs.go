package main

// RepoStruct - struct for storing repository data retrieved from API requests
type RepoStruct struct {
	Name        string `json:"name"`
	Description string `json:"desc"`
	URL         string `json:"url"`
	Stars       int    `json:"stars"`
	Forks       int    `json:"forks"`
	Language    string `json:"lang"`
	Archived    bool   `json:"archived"`
}

// GHStats - storing stats about my currently public GitHub repositories
type GHStats struct {
	LangUse          map[string]int `json:"language_use"`
	TotalPublicRepos int            `json:"total_repos"`
	TotalStars       int            `json:"total_stars"`
	TotalForks       int            `json:"total_forks"`
}

// Collection - struct for storing data about photo collections
type Collection struct {
	Key         string `json:"key"`
	Created     string `json:"date_created"`
	Description string `json:"description"`
	PreviewURL  string `json:"preview_url"`
}

// ImageObject - holds the info about one individual image
type ImageObject struct {
	CompressedURL string `json:"compressed_url"`
	FullURL       string `json:"full_url"`
	FullResMIB    string `json:"full_res_file_size_mib"`
}
