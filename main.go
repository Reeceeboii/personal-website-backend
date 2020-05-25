package main

import (
	"compress/gzip"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	js "github.com/buger/jsonparser"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	// aws
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// HTTP client for outbound requests
var client *http.Client

// AWS session
var awsSesh *session.Session

// S3 specific session
var s3svc *s3.S3

const gitHubRepoURL = "https://api.github.com/user/repos?visibility=public&affiliation=owner"

/*
	Generate the root HTML for any requests that land there. // TODO this needs to be updated when I get a minute.
	At the minute it's just an (almost) direct port from my current backend.
*/
func root(writer http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("root.gohtml")
	if err != nil {
		log.Fatalf("Error creating template: %+v", err)
	}
	type data struct{}

	tmpl.Execute(writer, data{})
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
			Stars:       int(stars),
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
		// if key already seen, increment its value, else initialise its value to 1
		if _, ok := langMap[language]; ok {
			langMap[language]++
		} else {
			langMap[language] = 1
		}
	})

	stats := GHStats{
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

func listCollections(writer http.ResponseWriter, r *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Content-Encoding", "gzip")

	bucket := os.Getenv("AWS_BUCKET_NAME")
	resp, err := s3svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: &bucket})
	if err != nil {
		log.Fatalf("Error listing items: %+v", err.Error())
	}

	var collectionsSlice []Collection
	// format each collection (folder i.e. has size of zero) and append to slice
	for _, item := range resp.Contents {
		var description string
		if *item.Size == 0 {
			if req, err := http.NewRequest("GET", formatPublicURL(strings.TrimSuffix(*item.Key, "/")+"/desc.json"), nil); err != nil {
				log.Fatalf("Error generating collection description get request: %+v", err)
			} else {
				if response, err := client.Do(req); err != nil {
					log.Fatalf("Error doing collection description get request: %+v", err)
				} else {
					defer response.Body.Close()
					parsed, _ := ioutil.ReadAll(response.Body)
					description, _ = js.GetString(parsed, "desc")
				}
			}

			tempCollection := Collection{
				Key:         strings.TrimSuffix(*item.Key, "/"),
				Created:     item.LastModified.Format("January 02, 2006"),
				Description: description,
			}
			collectionsSlice = append(collectionsSlice, tempCollection)
			formatPublicURL(strings.TrimSuffix(*item.Key, "/"))
		}
	}

	// gzip and send
	gz := gzip.NewWriter(writer)
	json.NewEncoder(gz).Encode(collectionsSlice)
	gz.Close()

}

// plug in S3 info and a key to create a publicly accessible URL for an S3 resource
func formatPublicURL(key string) string {
	return "https://s3." + os.Getenv("AWS_REGION") + ".amazonaws.com/" + os.Getenv("AWS_BUCKET_NAME") + "/" + key
}

// do some setup before server spins up
func init() {
	// load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No env")
	}

	// setup HTTP client
	client = &http.Client{
		// 10 second timeout stops silent failures if any external endpoints never respond
		Timeout: time.Second * 10,
	}

	// create a new AWS session
	awsSesh, _ = session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})

	// create a new S3 specific session
	s3svc = s3.New(awsSesh)
}

func main() {
	const base = "/api"
	router := mux.NewRouter().StrictSlash(true) // create new Mux router

	// root of server
	router.HandleFunc("/", root).Methods("GET")

	// GitHub routes
	router.HandleFunc(base+"/github/repos", repos).Methods("GET")
	router.HandleFunc(base+"/github/repo-stats", repoStats).Methods("GET")

	// photography routes
	router.HandleFunc(base+"/photography/list-collections", listCollections).Methods("GET")

	// start server and listen on port
	log.Println("Listening!")
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}
