package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	js "github.com/buger/jsonparser"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const gitHubRepoURL = "https://api.github.com/user/repos?visibility=public&affiliation=owner"

//const githubRefreshTImeout = strconv.Atoi(Getenv("GITHUB_REFRESH_SECONDS"))

// concurrency safe mutex lock wrapper around a slice of RepoStruct instances
type MutexRepositoryWrapper struct {
	mutex        sync.Mutex
	repositories []RepoStruct
}

var mutexData = MutexRepositoryWrapper{}

// lock the mutex, update the repositories slice, unlock the mutex
func (mutexRepos *MutexRepositoryWrapper) updateData(force bool) {

	// nested function used to make the API call
	update := func(mutexRepos *MutexRepositoryWrapper) {
		log.Println("Updating data from GitHub...")
		body := getRepos()
		mutexRepos.repositories = []RepoStruct{}

		js.ArrayEach(body, func(value []byte, dataType js.ValueType, offset int, err error) {
			name, _ := js.GetString(value, "name")
			desc, _ := js.GetString(value, "description")
			url, _ := js.GetString(value, "html_url")
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
			}
			mutexRepos.repositories = append(mutexRepos.repositories, tmpRepo)
		})
		log.Println("Update complete!")
	}

	// read out the refresh rate from the environment
	refreshRate, err := time.ParseDuration(fmt.Sprintf("%ss", os.Getenv("GITHUB_REFRESH_SECONDS")))
	if err != nil {
		log.Fatal(err)
	}

	// if the function is being forced, we do not need to wait for the ticker
	if force {
		log.Println("GITHUB UPDATE BEING FORCED")
		mutexRepos.mutex.Lock()
		update(mutexRepos)
		mutexRepos.mutex.Unlock()
	} else {
		// for every tick in <GITHUB_REFRESH_SECONDS>, lock the mutex and update the data
		for range time.Tick(refreshRate) {
			mutexRepos.mutex.Lock()
			update(mutexRepos)
			mutexRepos.mutex.Unlock()
		}
	}
}

// lock the mutex, read out the repositories and unlock mutex after returning data
func (mutexRepos *MutexRepositoryWrapper) getRepositories() []RepoStruct {
	mutexRepos.mutex.Lock()
	defer mutexRepos.mutex.Unlock()
	return mutexRepos.repositories
}

// Grab repo details from GitHub and parse response body
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

// Return formatted information about all my current public repos.
func repos(writer http.ResponseWriter, r *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Content-Encoding", "gzip")

	// gzip and send
	gz := gzip.NewWriter(writer)
	defer gz.Close()
	json.NewEncoder(gz).Encode(mutexData.getRepositories())
}

// Provide stats about used languages and other such numbers from my public GitHub repositories
func repoStats(writer http.ResponseWriter, r *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Content-Encoding", "gzip")

	// map of: <language name: x><occurrences of x as dominant language in public repos>
	langMap := make(map[string]int)

	var totalForks, totalStars, total int

	for _, repo := range mutexData.getRepositories() {
		total++
		totalForks += repo.Forks
		totalStars += repo.Stars
		// if key already seen, increment its value, else initialise its value to 1
		if _, ok := langMap[repo.Language]; ok {
			langMap[repo.Language]++
		} else {
			langMap[repo.Language] = 1
		}
	}

	stats := GHStats{
		LangUse:          langMap,
		TotalPublicRepos: total,
		TotalForks:       totalForks,
		TotalStars:       totalStars,
	}

	// gzip and send
	gz := gzip.NewWriter(writer)
	defer gz.Close()
	json.NewEncoder(gz).Encode(stats)
}
