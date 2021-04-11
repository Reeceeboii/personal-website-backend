package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	// aws
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

// StaticInformation - holds various pieces of non-changing data about the current server release
type StaticInformation struct {
	GoRuntime      string
	AWSSDKName     string
	AWSSDKVersion  string
	ServerBootTime time.Time
}

// CumulativeServerInformation - holds pieces of data that change as time goes on
type CumulativeServerInformation struct {
	callsToGitHubAPI         uint64
	getReposHits             uint64
	getRepoStatsHits         uint64
	listCollectionHits       uint64
	getCollectionContentHits uint64
}

// set up the server's data structs
var StaticInfo = StaticInformation{}
var CumulativeInfo = CumulativeServerInformation{}

// HTTP client for outbound requests
var client *http.Client

// AWS session
var awsSesh *session.Session

// S3 specific session
var s3svc *s3.S3

// Redirect any landing page requests
func root(writer http.ResponseWriter, r *http.Request) {
	http.Redirect(writer, r, "https://reecemercer.dev", 303)
}

// 404 route
func fourOhFour(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not found", http.StatusNotFound)
}

// Do some setup before server spins up
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

// Simply output a log of each incoming request
func middlewareLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Println("-> INCOMING", request.Method, "REQUEST TO", request.RequestURI)
		// Call the next handler
		next.ServeHTTP(writer, request)
	})
}

// Enable CORS headers on all responses
func middlewareCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// insert CORS header into ResponseWriter
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		// Call the next handler
		next.ServeHTTP(writer, request)
	})
}

// middleware to handle cache control headers on server responses
func middlewareCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// insert cache control header into ResponseWriter
		writer.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", int(time.Hour.Seconds())))
		// call the next handler
		next.ServeHTTP(writer, request)
	})
}

func main() {
	const base = "/api"

	// create new Mux router
	router := mux.NewRouter().StrictSlash(true)

	//apply logging, CORS and caching middleware
	router.Use(middlewareLogger)
	router.Use(middlewareCORS)
	router.Use(middlewareCache)

	// root of server
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

	// store some of the important data about the server when it first starts up
	StaticInfo.GoRuntime = runtime.Version()
	StaticInfo.AWSSDKName = aws.SDKName
	StaticInfo.AWSSDKVersion = aws.SDKVersion
	StaticInfo.ServerBootTime = time.Now()

	// initialise the server to get it ready to send emails
	InitEmailData()
	// send the first server boot email out
	SendServerBootEmail()
	// start the periodic outbound emails
	go EmailJob()

	if os.Getenv("GO_ENV") == "development" {
		// get the executable's current working directory
		workingDirectory, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Working dir: %s\n", workingDirectory)
	}

	/*
	  Start periodic updates from GitHub. We can force one starter update through when the server
	  first executes else we'd be left without data until the first time the ticker fires
	*/
	mutexData.updateData(true)
	go mutexData.updateData(false)

	// start server and listen on port
	log.Println("Listening!")
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}
