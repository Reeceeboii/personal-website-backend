// Kicks off everything

package main

import (
	"github.com/Reeceeboii/personal-website-backend/pkg/server"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
)

func main() {
	// load in environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("No env")
	}

	server.BackendServer = server.NewServer()

	// set up the logger
	log.SetFlags(0)
	log.SetOutput(server.BackendServer.Logger)

	// listen and serve
	log.Println("Listening!")
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), server.BackendServer.Router))
}
