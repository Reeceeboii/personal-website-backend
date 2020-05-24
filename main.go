package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

// create a new HTTP client so we can send outbound HTTP requests
var client = &http.Client{
	// 10 second timeout stops silent failures if any external endpoints never respond
	Timeout: time.Second * 10,
}

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

func data(writer http.ResponseWriter, r *http.Request) {
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

func main() {
	router := mux.NewRouter().StrictSlash(true) // create new Mux router
	router.HandleFunc("/", root).Methods("GET")
	router.HandleFunc("/data", data).Methods("GET")

	// start server and listen on port
	log.Println("Listening!")
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}
