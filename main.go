package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var appFlickrOAuth = new(FlickrOAuth)
var rootDirectory = flag.String("dir", "", "The base directory where your sets/photos will be downloaded.")

func main() {

	flag.Parse()

	if *rootDirectory == "" {
		fmt.Println("You must specify a root directory using -dir")
		return
	}

	l := createLogger()

	appFlickrOAuth := checkForExistingOAuthCredentials()

	if appFlickrOAuth.OAuthToken != "" {
		logMessage(l, "Using credentials for user: "+appFlickrOAuth.Username, true)
	} else {
		appFlickrOAuth = doOAuthSetup()
		if appFlickrOAuth.OAuthToken == "" {
			logMessage(l, "Could not get OAuth token setup.", true)
			return
		}
	}

	// Get the sets, ordered by created date
	sets := getSets(appFlickrOAuth)

	for _, v := range sets.SetContainer.Sets {

		// Create the directory for this set with the set's created
		// date as the prefix so the directories are ordered the same way
		// flickr orders the sets
		t := time.Unix(int64(v.DateCreated), 0)
		format := "20060102"
		cleanTitle := cleanTitle(v.Title)
		dir := filepath.Join(*rootDirectory, t.Format(format)+" "+cleanTitle)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			panic(err)
		}

		// Skip sets that already have all their files downloaded
		existingFiles, _ := ioutil.ReadDir(dir)
		if len(existingFiles) == (v.Photos + v.Videos + 1) {
			logMessage(l, "Skipping set: `"+v.Title+"'. Found "+strconv.Itoa(len(existingFiles))+" existing files.", false)
			continue
		}

		logMessage(l, "Processing set: `"+v.Title+"'.", false)

		metadataFile := filepath.Join(dir, "metadata.json")
		var metadata Metadata

		// Read the existing metadata, or create a new struct if none is found,
		// so we can pick up where we left off
		if _, err := os.Stat(metadataFile); !os.IsNotExist(err) {
			existingMetadata, _ := ioutil.ReadFile(metadataFile)
			err = json.Unmarshal(existingMetadata, &metadata)
		} else {
			metadata = Metadata{Photos: []PhotoMetadata{}, SetId: v.Id}
		}

		// Get all the photos for this set and loop over them
		photos := getPhotosForSet(appFlickrOAuth, v)
		for _, vv := range photos {

			originalUrl := getOriginalSizeUrl(appFlickrOAuth, vv)
			if originalUrl == "" {
				logMessage(l, "Could not get original size for photo: `"+vv.Title+"' ("+vv.Id+")", false)
			} else {

				// Create the file name from the url
				fileName := getFileNameFromUrl(originalUrl)
				fullPath := filepath.Join(dir, fileName)

				// Skip files that exist
				if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
					logMessage(l, "Photo existed at "+fullPath+", skipping.", false)
					continue
				}

				// Save photo to disk
				savePhotoToFile(appFlickrOAuth, originalUrl, fullPath)

				// Add the photos metadata to the list and write the metadata file out
				p := PhotoMetadata{PhotoId: vv.Id, Title: vv.Title, Filename: fileName}
				slice := append(metadata.Photos, p)
				metadata.Photos = slice
				metadataBytes, _ := json.Marshal(metadata)
				ioutil.WriteFile(metadataFile, metadataBytes, 0755)

				logMessage(l, "Saved photo `"+vv.Title+"' to "+fullPath, false)
			}
		}
	}
}

func cleanTitle(title string) string {

	invalidChars := []string{"\\", "/", ":", ">", "<", "?", "\"", "|", "*"}

	for _, char := range invalidChars {
		title = strings.Replace(title, char, "", -1)
	}

	return title
}

// Extracts the photo file name from the url
func getFileNameFromUrl(url string) string {

	index := strings.LastIndex(url, "/")
	if index == -1 {
		fmt.Println("No / in " + url)
		return ""
	}
	return url[index+1:]
}

// Sets up the logger for the app
func createLogger() *log.Logger {

	t := time.Now()
	format := "20060102"

	filePart := t.Format(format) + "-"
	filePart += strconv.Itoa(t.Hour())
	filePart += strconv.Itoa(t.Minute())
	filePart += strconv.Itoa(t.Second())

	logDir := filepath.Join(*rootDirectory, "logs")
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		panic(err)
	}

	filePath := filepath.Join(logDir, "fsync-"+filePart+".log")

	fi, _ := os.Create(filePath)
	l := log.New(fi, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	return l
}

// Logs a message and optionally echos it to stdout
func logMessage(l *log.Logger, message string, echo bool) {

	l.Println(message)
	if echo {
		fmt.Println(message)
	}
}

type Metadata struct {
	SetId  string
	Photos []PhotoMetadata
}

// Photo metadata struct
type PhotoMetadata struct {
	PhotoId  string
	Title    string
	Filename string
}
