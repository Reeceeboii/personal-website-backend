package main

import (
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	js "github.com/buger/jsonparser"
)

/*
   Grab repo details from GitHub and parse reponse body
*/
func getRepos() (body []byte) {
	req, err := http.NewRequest("GET", gitHubRepoURL, nil)
	if err != nil {
		log.Fatalf("Error generating get repo request: %+v", err)
	}

	req.Header.Set("Authorization", "Token "+os.Getenv("GITHUB_API_TOKEN")) // set GH auth header

	response, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error making get repo request: %+v", err.Error())
	}
	defer response.Body.Close()
	parsedBody, _ := ioutil.ReadAll(response.Body)
	return parsedBody
}

/*
   Return formatted information about all my current public repos.
   This uses the GitHub API
*/
func repos(writer http.ResponseWriter, r *http.Request) {
	body := getRepos()
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Content-Encoding", "gzip")

	// construct custom repo struct for every repo returned from API and append to slice
	repoStructSlice := []RepoStruct{}
	js.ArrayEach(body, func(value []byte, dataType js.ValueType, offset int, err error) {
		name, _ := js.GetString(value, "name")
		desc, _ := js.GetString(value, "description")
		url, _ := js.GetString(value, "html_url")
		httpClone, _ := js.GetString(value, "clone_url")
		sshClone, _ := js.GetString(value, "ssh_url")
		stars, _ := js.GetInt(value, "stargazers_count")
		forks, _ := js.GetInt(value, "forks_count")
		language, _ := js.GetString(value, "language")
		archived, _ := js.GetBoolean(value, "archived")

		tmpRepo := RepoStruct{
			Name:        name,
			Description: desc,
			URL:         url,
			Stars:       int(stars), // GetInt provides int64
			Forks:       int(forks),
			Language:    language,
			Archived:    archived,
			Clones: CloneSources{
				HTTP: httpClone,
				SSH:  sshClone,
			},
		}
		repoStructSlice = append(repoStructSlice, tmpRepo)
	})

	// gzip and send
	gz := gzip.NewWriter(writer)
	json.NewEncoder(gz).Encode(repoStructSlice)
	gz.Close()
}
