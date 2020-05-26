package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	js "github.com/buger/jsonparser"
)

/*
   Returns the names of all collections, their creation dates and their descriptions
*/
func listCollections(writer http.ResponseWriter, r *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Content-Encoding", "gzip")

	var collectionsSlice []Collection
	// format each collection (folder i.e. has size of zero) and append to slice
	for _, item := range listBucket() {
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
				Created:     item.LastModified.Format("January 02, 2006"), // format the date (why on earth does Go do it like this?)
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

/*
   Getting links for the images from a single collection - both compressed and original.
   This takes one query string (c) - the name of the collection that the links are to be generated for.
*/
func getCollectionContents(writer http.ResponseWriter, r *http.Request) {
	if collectionName := r.URL.Query().Get("c"); collectionName != "" {
		fmt.Println(collectionName)
	}

}

// plug in S3 info and a key to create a publicly accessible URL for an S3 resource
func formatPublicURL(key string) string {
	return "https://s3." + os.Getenv("AWS_REGION") + ".amazonaws.com/" + os.Getenv("AWS_BUCKET_NAME") + "/" + key
}
