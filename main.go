package main

import (
	"compress/gzip"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
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
	tmpl, err := template.ParseFiles("./views/root.gohtml")
	if err != nil {
		log.Fatalf("Error creating template: %+v", err)
	}

	var data struct{}
	tmpl.Execute(writer, data)
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

/*
   Return a slice of S3 object pointers representing every object in the entire bucket
*/
func listBucket() []*s3.Object {
	bucket := os.Getenv("AWS_BUCKET_NAME")
	resp, err := s3svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: &bucket})
	if err != nil {
		log.Fatalf("Error listing items: %+v", err.Error())
	}
	return resp.Contents
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

	// root of server - serve the landing page
	router.HandleFunc("/", root).Methods("GET")

	// GitHub API routes
	router.HandleFunc(base+"/github/repos", repos).Methods("GET")
	router.HandleFunc(base+"/github/repo-stats", repoStats).Methods("GET")

	// photography routes
	router.HandleFunc(base+"/photos/list-collections", listCollections).Methods("GET")
	router.HandleFunc(base+"/photos/get-contents", getCollectionContents).Methods("GET")

	// start server and listen on port
	log.Println("Listening!")
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}
