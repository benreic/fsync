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
var setId = flag.String("setId", "", "Only process a single set; applies to audit and actual processing")
var forceProcessing = flag.Bool("force", false, "Force processing of each set; don't skip sets even if file counts match")
var auditOnly = flag.Bool("audit", false, "Compares existing media with the media on Flickr and displays the differences")
var Flogger *log.Logger

var setMetadataFileName = "metadata.json"

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

		if *setId != "" && v.Id != *setId {
			continue
		}

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

		existingFiles, _ := ioutil.ReadDir(dir)

		metadataFile := filepath.Join(dir, setMetadataFileName)
		var metadata Metadata

		// Read the existing metadata, or create a new struct if none is found,
		// so we can pick up where we left off
		if fileExists(metadataFile) {
			existingMetadata, _ := ioutil.ReadFile(metadataFile)
			err = json.Unmarshal(existingMetadata, &metadata)
		} else {
			metadata = Metadata{Photos: []PhotoMetadata{}, SetId: v.Id}
		}

		if *auditOnly == true {

			auditSet(existingFiles, &metadata, photos, v, metadataFile, dir)
			continue
		}

		if *forceProcessing != true {
			// Skip sets that already have all their files downloaded
			if len(existingFiles) == (len(photos)+1) && len(photos) == len(metadata.Photos) {
				logMessage(fmt.Sprintf("Skipping set: `%v'. Found %v existing files.", v.Title, strconv.Itoa(len(existingFiles))), false)
				continue
			}
			
			logMessage(fmt.Sprintf("Processing set: `%v'. Found %v existing files on disk, %v files in metadata, and %v files on Flickr.", v.Title, strconv.Itoa(len(existingFiles)), strconv.Itoa(len(metadata.Photos)), strconv.Itoa(len(photos))), false)
		} else {
			logMessage(fmt.Sprintf("Force processing set: `%v'", v.Title), false)
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

					// Regardless of whether the file exists on disk, make 
					// sure its gone from the metadata
					removeFileFromMetadata(vv, fileName, &metadata, metadataFile)
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
			saveMetadataToFile(vv, fileName, &metadata, metadataFile)
			logMessage(fmt.Sprintf("Saved %v `%v' to %v.", mediaType, vv.Title, fullPath), false)

		}
	}
}

/*
Loop through the photos in the set. See if each media exists in the metadata. Keep track of photos
that don't exist in the metadata, these need to be downloaded.

Loop through the media in the metadata. Any that don't exist in the set should be deleted and removed from the metadata.

Loop through the file and make sure they are all in the metadata.

*/
func auditSet(existingFiles []os.FileInfo, metadata *Metadata, photos map[string]Photo, set Photoset, metadataFile string, setDir string) {

	logMessage(fmt.Sprintf("Auditing set: `%v'", set.Title), true)

	// Convert the metadata into a map for ease of use
	photoIdMap := map[string]PhotoMetadata{}
	fileNameMap := map[string]PhotoMetadata{}

	for _, pm := range metadata.Photos {
		photoIdMap[pm.PhotoId] = pm
		fileNameMap[pm.Filename] = pm
	}

	// Find photos that don't exist on disk and need to be downloaded
	for photoId, photo := range photos {

		_, valueExists := photoIdMap[photoId]
		if valueExists == false {
			logMessage(fmt.Sprintf("Photo Id  `%v' (%v) needs to be downloaded.", photoId, photo.Title), true)
		}
	}

	// Find photos that exist on disk, but not in Flickr, they need to be deleted.
	for photoId, pm := range photoIdMap {

		if _, ok := photos[photoId]; ! ok {
			logMessage(fmt.Sprintf("Photo id `%v' (%v) needs to be deleted.", photoId, pm.Title), true)
		}
	}

	// Find photos on disk that are not in the metadata
	for _, fi := range existingFiles {
		if fi.Name() == setMetadataFileName {
			continue
		}
		_, valueExists := fileNameMap[fi.Name()]
		if valueExists == false {
			logMessage(fmt.Sprintf("File exists on disk, but not in metadata. This is a bug.: `%v'.", fi.Name()), true)
		}
	}

	// Find photos in metadata that are not on disk
	for fileName, _ := range fileNameMap {
		if fileName == setMetadataFileName {
			continue
		}

		// make the full file path from the filename
		fullFileName := filepath.Join(setDir, fileName)
		if ! fileExists(fullFileName) {
			logMessage(fmt.Sprintf("File exists in metadata, but not on disk. The file was either deleted or never saved correctly. This is a bug.: `%v'.", fullFileName), true)
		}
	}
}

func saveMetadataToFile(media Photo, fileName string, metadata *Metadata, metadataFile string) {

	p := PhotoMetadata{PhotoId: media.Id, Title: media.Title, Filename: fileName}

	// See if there is an existing entry for this photo
	// update the metadata if there is
	var foundPhoto = false
	for index, photo := range metadata.Photos {
		if photo.PhotoId == p.PhotoId {
			metadata.Photos[index].Title = p.Title
			metadata.Photos[index].Filename = p.Filename
			foundPhoto = true
			logMessage("Updating existing entry in metadata.", false)
			break
		}
	}

	// Didn't find an existing one, so add to the metadata
	if ! foundPhoto {
		slice := append(metadata.Photos, p)
		metadata.Photos = slice
	}

	// serialize it and save
	metadataBytes, _ := json.Marshal(metadata)
	ioutil.WriteFile(metadataFile, metadataBytes, 0755)
}

func removeFileFromMetadata(media Photo, fileName string, metadata *Metadata, metadataFile string) {

	var newListOfMedia = []PhotoMetadata{ }
	for _, photo := range metadata.Photos { 
		if photo.Filename != fileName {
			newListOfMedia = append(newListOfMedia, photo)
		} else {
			logMessage(fmt.Sprintf("Removing filename `%v' from the metadata.", fileName), true)
		}
	}

	metadata.Photos = newListOfMedia
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
	format := "20060102-150405"

	filePart := t.Format(format)

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
