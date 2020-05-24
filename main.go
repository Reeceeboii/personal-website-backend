package main

import (
	"encoding/json"
	"fmt"
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
	Return formatted information about all my current public repos.
	This uses the GitHub API
*/
func repos(writer http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequest("GET", "https://api.github.com/user/repos?visibility=public&affiliation=owner", nil)
	if err != nil {
		log.Fatalf("Error generating get repo request: %+v", err)
	}

	req.Header.Set("Authorization", "Token "+os.Getenv("GITHUB_API_TOKEN")) // set GH auth header
	writer.Header().Set("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error making get repo request: %+v", err.Error())
	}
	body, err := ioutil.ReadAll(response.Body)

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

	// marshal struct data to JSON string and return to client
	formatted, err := json.Marshal(repoStructArr)
	if err != nil {
		log.Fatalf("Error marshalling repo data: %+v", err.Error())
	}
	fmt.Fprintf(writer, string(formatted))
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
	router.HandleFunc("/", root).Methods("GET")
	router.HandleFunc(base+"/github/repos", repos).Methods("GET")

	// start server and listen on port
	log.Println("Listening!")
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}
