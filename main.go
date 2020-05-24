package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/Reeceeboii/personal-website-backend"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger"
)

// create a new HTTP client so we can send outbound HTTP requests
var client = &http.Client{
	// 10 second timeout stops silent failures if any external endpoints never respond
	Timeout: time.Second * 10,
}

func root(writer http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		log.Fatalf("Error generating request. %+v", err)

	}
	req.Header.Set("Authorization", "Token "+os.Getenv("GITHUB_API_TOKEN")) // set GH auth header
	writer.Header().Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error making request: %+v", err.Error())
	}
	body, err := ioutil.ReadAll(response.Body)
	fmt.Fprintf(writer, string(body))
}

func init() {
	// load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No env")
	}
}

// @title Swagger Example API
// @version 1.0
// @description This is a sample server Petstore server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host petstore.swagger.io
// @BasePath /v2
func main() {
	router := mux.NewRouter().StrictSlash(true) // create new Mux router
	router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("http://localhost:5000/swagger/doc.json"), //The url pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("#swagger-ui"),
	))

	router.HandleFunc("/", root).Methods("GET")
	// start server and listen on port
	log.Println("Listening!")
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}
