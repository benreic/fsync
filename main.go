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
var Flogger *log.Logger

func main() {

	flag.Parse()

	Flogger = createLogger()

	if *rootDirectory == "" {
		fmt.Println("You must specify a root directory using -dir")
		return
	}

	appFlickrOAuth := checkForExistingOAuthCredentials()

	if appFlickrOAuth.OAuthToken != "" {
		logMessage(fmt.Sprintf("Using credentials for user: %v", appFlickrOAuth.Username), true)
	} else {
		appFlickrOAuth = doOAuthSetup()
		if appFlickrOAuth.OAuthToken == "" {
			logMessage("Could not get OAuth token setup.", true)
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

		// Get all the photos for this set and loop over them
		photos := getPhotosForSet(appFlickrOAuth, v)

		// Skip sets that already have all their files downloaded
		existingFiles, _ := ioutil.ReadDir(dir)
		if len(existingFiles) == len(photos) + 1 {
			logMessage(fmt.Sprintf("Skipping set: `%v'. Found %v existing files.", v.Title, strconv.Itoa(len(existingFiles))), false)
			continue
		}

		logMessage(fmt.Sprintf("Processing set: `%v'", v.Title), false)

		metadataFile := filepath.Join(dir, "metadata.json")
		var metadata Metadata

		// Read the existing metadata, or create a new struct if none is found,
		// so we can pick up where we left off
		if fileExists(metadataFile) {
			existingMetadata, _ := ioutil.ReadFile(metadataFile)
			err = json.Unmarshal(existingMetadata, &metadata)
		} else {
			metadata = Metadata{Photos: []PhotoMetadata{}, SetId: v.Id}
		}


		var fullPath string
		var fileName string
		var sourceUrl string
		var mediaType string
		for _, vv := range photos {

			// Get the photo and video url (if one exists)
			photoUrl, videoUrl := getOriginalSizeUrl(appFlickrOAuth, vv)

			if videoUrl != "" {

				// If we erroneously downloaded a video as a photo
				// before, lets delete it before getting the actual video
				if photoUrl != "" {

					// Create the photo file name from the url
					fileName = getFileNameFromUrl(photoUrl)
					fullPath = filepath.Join(dir, fileName)

					// Remove the photo, if we found one
					if fileExists(fullPath) {
						os.Remove(fullPath)
						logMessage(fmt.Sprintf("Deleted the video's photo located at: %v.", fullPath), false)
					}
				}

				fileName = vv.Id + ".mov"
				sourceUrl = videoUrl
				mediaType = "video"
				
			} else if photoUrl != "" {

				// Create the file name from the url
				fileName = getFileNameFromUrl(photoUrl)
				sourceUrl = photoUrl
				mediaType = "photo"
				
			} else {

				logMessage(fmt.Sprintf("Could not get original size for media: `%v' (%v). Skipping media for now.", vv.Title, vv.Id), false)
				continue
			}


			fullPath = filepath.Join(dir, fileName)

			// Skip files that exist
			if fileExists(fullPath) {
				logMessage(fmt.Sprintf("Media existed at %v. Skipping.", fullPath), false)
				continue
			}

			// Save video to disk
			saveUrlToFile(appFlickrOAuth, sourceUrl, fullPath)

			// Add the photos metadata to the list and write the metadata file out
			saveMediaMetadata(vv, fileName, metadata, metadataFile)

			logMessage(fmt.Sprintf("Saved %v `%v' to %v.", mediaType, vv.Title, fullPath), false)


		}
	}
}


func saveMediaMetadata(media Photo, fileName string, metadata Metadata, metadataFile string) {

	p := PhotoMetadata{PhotoId: media.Id, Title: media.Title, Filename: fileName}
	slice := append(metadata.Photos, p)
	metadata.Photos = slice
	metadataBytes, _ := json.Marshal(metadata)
	ioutil.WriteFile(metadataFile, metadataBytes, 0755)
}


func fileExists(fullPath string) bool {

	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		return true
	}

	return false

}


func saveUrlToFile(flickrOauth FlickrOAuth, url string, fullPath string) {

	var err error
	var body []byte

	body, err = makeGetRequest(url)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(fullPath, body, 0644)
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
func logMessage(message string, echo bool) {

	Flogger.Println(message)
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
