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

	"github.com/aws/aws-sdk-go/service/s3"
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

			collectionsSlice = append(collectionsSlice, Collection{
				Key:         strings.TrimSuffix(*item.Key, "/"),
				Created:     item.LastModified.Format("January 02, 2006"), // format the date (why on earth does Go do it like this?)
				Description: description,
				PreviewURL:  getCollectionPreviewLink(*item.Key),
			})
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
	return strings.ReplaceAll("https://s3." + os.Getenv("AWS_REGION") + ".amazonaws.com/" + os.Getenv("AWS_BUCKET_NAME") + "/" + key, " ", "+")
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

/*
  Given the name of a photo collection (S3 folder object), return the URL for its preview image.
*/
func getCollectionPreviewLink(collectionName string) string {
	// make sure folder has appended slash
	if !strings.Contains(collectionName, "/") {
		collectionName = collectionName + "/"
	}

	prefix := collectionName + "_preview_"
	bucket := os.Getenv("AWS_BUCKET_NAME")
	resp, err := s3svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: &bucket, Prefix: &prefix})
	if err != nil {
		log.Fatalf("Error listing items: %+v", err.Error())
	} else {
		for _, item := range resp.Contents {
			if strings.Contains(*item.Key, "compressed") {
				return formatPublicURL(*item.Key)
			}
		}
	}
	return ""
}
