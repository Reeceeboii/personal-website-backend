package main

import (
	"compress/gzip"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	js "github.com/buger/jsonparser"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

// create a new HTTP client so we can send outbound HTTP requests
var client = &http.Client{
	// 10 second timeout stops silent failures if any external endpoints never respond
	Timeout: time.Second * 10,
}

const gitHubRepoURL = "https://api.github.com/user/repos?visibility=public&affiliation=owner"

// struct for storing repository data retrieved from API requests
type repoStruct struct {
	Name        string       `json:"name"`
	Description string       `json:"desc"`
	URL         string       `json:"url"`
	Stars       int          `json:"stars"`
	Forks       int          `json:"forks"`
	Language    string       `json:"lang"`
	Archived    bool         `json:"archived"`
	Clones      cloneSources `json:"clones"`
}

// storing the different ways the repositories can be cloned locally
type cloneSources struct {
	HTTP string `json:"http_clone"`
	SSH  string `json:"ssh_clone"`
}

type ghStats struct {
	LangUse          map[string]int `json:"language_use"`
	TotalPublicRepos int            `json:"total_repos"`
	TotalStars       int            `json:"total_stars"`
	TotalForks       int            `json:"total_forks"`
}

/*
	Generate the root HTML for any requests that land there. // TODO this needs to be updated when I get a minute.
	At the minute it's just an (almost) direct port from my current backend.
*/
func root(writer http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("root.gohtml")
	if err != nil {
		log.Fatalf("Error creating template: %+v", err)
	}
	date := time.Now().Month().String() + " " + strconv.Itoa(time.Now().Day()) + " " + strconv.Itoa(time.Now().Year())
	data := struct {
		Time string
		Path string
	}{
		Time: date,
		Path: r.URL.Path,
	}
	tmpl.Execute(writer, data)
}

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
	defer response.Body.Close()
	if err != nil {
		log.Fatalf("Error making get repo request: %+v", err.Error())
	}
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
	repoStructArr := []repoStruct{}
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

		tmpRepo := repoStruct{
			Name:        name,
			Description: desc,
			URL:         url,
			Stars:       int(stars),
			Forks:       int(forks),
			Language:    language,
			Archived:    archived,
			Clones: cloneSources{
				HTTP: httpClone,
				SSH:  sshClone,
			},
		}
		repoStructArr = append(repoStructArr, tmpRepo)
	})

	// gzip and send
	gz := gzip.NewWriter(writer)
	json.NewEncoder(gz).Encode(repoStructArr)
	gz.Close()
}

/*
	Provide stats about used languages and other such numbers from my public GitHub repositories
*/
func repoStats(writer http.ResponseWriter, r *http.Request) {
	body := getRepos()
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Content-Encoding", "gzip")

	// map of: <language name: x><occurrences of x as dominant language in public repos>
	langMap := make(map[string]int)

	var totalForks, totalStars, total int

	js.ArrayEach(body, func(value []byte, dataType js.ValueType, offset int, err error) {
		language, _ := js.GetString(value, "language")
		stars, _ := js.GetInt(value, "stargazers_count")
		forks, _ := js.GetInt(value, "forks_count")

		total++
		totalForks += int(forks)
		totalStars += int(stars)
		// if language already seen, increment its value, else initialise its value to 1
		if _, ok := langMap[language]; ok {
			langMap[language]++
		} else {
			langMap[language] = 1
		}

	})

	stats := ghStats{
		LangUse:          langMap,
		TotalPublicRepos: total,
		TotalForks:       totalForks,
		TotalStars:       totalStars,
	}

	// gzip and send
	gz := gzip.NewWriter(writer)
	json.NewEncoder(gz).Encode(stats)
	gz.Close()
}

func init() {
	// load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No env")
	}
}

func main() {
	const base = "/api"
	router := mux.NewRouter().StrictSlash(true) // create new Mux router

	// root of server
	router.HandleFunc("/", root).Methods("GET")

	// GitHub routes
	router.HandleFunc(base+"/github/repos", repos).Methods("GET")
	router.HandleFunc(base+"/github/repo-stats", repoStats).Methods("GET")

	// start server and listen on port
	log.Println("Listening!")
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}
