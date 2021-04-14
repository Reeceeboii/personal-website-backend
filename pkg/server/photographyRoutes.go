package server

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/s3"
	js "github.com/buger/jsonparser"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

// Stores data about photo collections
type Collection struct {
	// a unique name for a given collection - i.e. 'Cats'
	Key string `json:"key"`
	// the date that the collection was created
	Created string `json:"date_created"`
	// a brief description of the collection
	Description string `json:"description"`
	// a url to a compressed preview image
	PreviewURL string `json:"preview_url"`
}

// Stores info about one individual image
type ImageObject struct {
	// the URL to the compressed version of the image
	CompressedURL string `json:"compressed_url"`
	// the URL to the full resolution version of the image
	FullURL string `json:"full_url"`
	// the size, in mebibytes, of the full resolution image
	FullResMIB string `json:"full_res_file_size_mib"`
}

// Returns the names of all collections, their creation dates and their descriptions
func ListCollections(writer http.ResponseWriter, _ *http.Request) {
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
				if response, err := BackendServer.HTTPClient.Do(req); err != nil {
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

	if err := gz.Close(); err != nil {
		log.Printf("Error closing Gzip encoder (listCollections) %s", err.Error())
	}
}

/*
   Getting links for the images from a single collection - both compressed and original.
   This takes one query string (c) - the name of the collection that the links are to be generated for.
*/
func GetCollectionContents(writer http.ResponseWriter, r *http.Request) {
	if collectionName := r.URL.Query().Get("c"); collectionName != "" {
		bucket := os.Getenv("AWS_BUCKET_NAME")

		// does the queried collection name actually exist in the bucket?
		match := false
		if resp, err := BackendServer.S3Session.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket: &bucket,
		}); err != nil {
			log.Fatalf("Error getting objects for collection: %+v", err.Error())
		} else {
			for _, item := range resp.Contents {
				if *item.Size == 0 {
					if collectionName+"/" == *item.Key {
						match = true
					}
				}
			}
		}

		// if the object doesn't actually exist
		if !match {
			writer.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(writer, html.EscapeString(collectionName)+" doesn't exist")
			return
		}

		// once we're here, we're certain that the collection does exist, and that means we're fine
		// to begin querying it for objects
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Content-Encoding", "gzip")
		prefix := collectionName + "/_"
		resp, err := BackendServer.S3Session.ListObjectsV2(&s3.ListObjectsV2Input{
			Bucket: &bucket,
			Prefix: &prefix,
		})
		if err != nil {
			log.Fatalf("Error getting objects for collection: %+v", err.Error())
		}
		// https://docs.aws.amazon.com/AmazonS3/latest/dev/ListingKeysUsingAPIs.html
		// since keys are always listed in UTF8 binary order I can just take them together
		// and am essentially guaranteed their correctness as pairs. This saves all the string
		// operations that I implemented in my previous main and is therefore inherently a lot faster
		item := 0
		var pairsForCollection []ImageObject
		for item < len(resp.Contents)-1 {
			var size float64
			// cast file size (bytes as int64) to float 64, and divide by 1024 twice to get KiB and MiB
			size = float64(*resp.Contents[item+1].Size)
			size /= 1024
			size /= 1024
			pairsForCollection = append(pairsForCollection, ImageObject{
				CompressedURL: formatPublicURL(*resp.Contents[item].Key),
				FullURL:       formatPublicURL(*resp.Contents[item+1].Key),
				FullResMIB:    fmt.Sprintf("%.2f", size), // round off to 2 d.p
			})
			item += 2
		}
		// gzip and send
		gz := gzip.NewWriter(writer)
		defer gz.Close()
		json.NewEncoder(gz).Encode(pairsForCollection)
	} else {
		writer.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(writer, "I have no idea what you want me to do.")
	}
}

// Return a slice of S3 object pointers representing every object in the entire bucket
func listBucket() []*s3.Object {
	bucket := os.Getenv("AWS_BUCKET_NAME")
	resp, err := BackendServer.S3Session.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: &bucket,
	})
	if err != nil {
		log.Fatalf("Error listing items: %+v", err.Error())
	}
	return resp.Contents
}

// plug in S3 info and a key to create a publicly accessible URL for an S3 resource
func formatPublicURL(key string) string {
	return strings.ReplaceAll("https://s3."+os.Getenv("AWS_REGION")+".amazonaws.com/"+os.Getenv("AWS_BUCKET_NAME")+"/"+key, " ", "+")
}

// Given the name of a photo collection (S3 folder object), return the URL for its preview image.
func getCollectionPreviewLink(collectionName string) string {
	// make sure folder has appended slash
	if !strings.Contains(collectionName, "/") {
		collectionName = collectionName + "/"
	}

	prefix := collectionName + "_preview_"
	bucket := os.Getenv("AWS_BUCKET_NAME")
	resp, err := BackendServer.S3Session.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &prefix,
	})
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
