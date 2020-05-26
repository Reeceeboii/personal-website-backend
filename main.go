package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

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

/*
   Generate the root HTML for any requests that land there. // TODO this needs to be updated when I get a minute.
   At the minute it's just an (almost) direct port from my current backend.
*/
func root(writer http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("./views/root.gohtml")
	if err != nil {
		log.Fatalf("Error creating template: %+v", err)
	}

  data := struct{
    HostedAt string
  }{
    HostedAt: "https://reecemercer-dev-backend.herokuapp.com/",
  }
	tmpl.Execute(writer, data)
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
