package main

import (
	"compress/gzip"
	"fmt"
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
	writer.Header().Add("Content-Type", "text/html")
	writer.Header().Set("Content-Encoding", "gzip")
	tmpl, err := template.ParseFiles("./views/root.gohtml")
	if err != nil {
		log.Fatalf("Error creating template: %+v", err)
	}

	// pass the remote URL to the templater to create visitable URLs on the root page
	data := struct {
		RemoteURL string
	}{
		RemoteURL: "https://reecemercer-dev-backend.herokuapp.com",
	}

	// setup gzip
	gz := gzip.NewWriter(writer)
	defer gz.Close()

	// generate templated HTML and send
	tmpl.Execute(gz, data)
}

/*
  404 route
*/
func fourOhFour(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./views/404.gohtml")
}

/*
  Do some setup before server spins up
*/
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

// custom logger
type logger struct{}

func (writer logger) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().UTC().Format("Jan _2 15:04:05.000000"), " [LOG] ", string(bytes))
}

/*
  Simply output a log of each incoming request
*/
func middlewareLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Println("-> INCOMING", request.Method, "REQUEST TO", request.RequestURI)
		// Call the next handler
		next.ServeHTTP(writer, request)
	})
}

/*
  Enable CORS headers on all responses
*/
func middlewareCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// insert CORS header into ResponseWriter
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		// Call the next handler
		next.ServeHTTP(writer, request)
	})
}

func main() {
	const base = "/api"

	// create new Mux router
	router := mux.NewRouter().StrictSlash(true)

	//apply logging and CORS middleware
	router.Use(middlewareLogger)
	router.Use(middlewareCORS)

	// root of server - serve the landing page
	router.HandleFunc("/", root).Methods("GET")

	// GitHub API routes
	router.HandleFunc(base+"/github/repos", repos).Methods("GET")
	router.HandleFunc(base+"/github/repo-stats", repoStats).Methods("GET")

	// photography routes
	router.HandleFunc(base+"/photos/list-collections", listCollections).Methods("GET")
	router.HandleFunc(base+"/photos/get-contents", getCollectionContents).Methods("GET")

	// and for any non matching routes, send 404 response back
	router.NotFoundHandler = http.HandlerFunc(fourOhFour)

	// setup logger
	log.SetFlags(0)
	log.SetOutput(new(logger))

	// start server and listen on port
	log.Println("Listening!")
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}
