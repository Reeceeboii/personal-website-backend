package main

// RepoStruct - struct for storing repository data retrieved from API requests
type RepoStruct struct {
	Name        string       `json:"name"`
	Description string       `json:"desc"`
	URL         string       `json:"url"`
	Stars       int          `json:"stars"`
	Forks       int          `json:"forks"`
	Language    string       `json:"lang"`
	Archived    bool         `json:"archived"`
	Clones      CloneSources `json:"clones"`
}

// CloneSources - storing the different ways the repositories can be cloned locally
type CloneSources struct {
	HTTP string `json:"http_clone"`
	SSH  string `json:"ssh_clone"`
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
}