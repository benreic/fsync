package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

/**
 * Main function to kick off processing of Flickr sets
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @return  void
**/

func processSets() {

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

	sets := determineSetsToProcess(appFlickrOAuth)

	for _, set := range sets {
		processSingleSet(appFlickrOAuth, set)
	}
}

/**
 * Figures out which sets to process based on command line params
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   FlickrOAuth   The oauth configuration
 * @return  []Photoset    The list of photosets to process
**/

func determineSetsToProcess(appFlickrOAuth FlickrOAuth) []Photoset {

	var sets []Photoset
	if !*onlyPhotosNotInSet {

		// Get the sets, ordered by created date
		flickrSets := getSets(appFlickrOAuth)

		for _, set := range flickrSets.SetContainer.Sets {

			if *setId != "" && set.Id != *setId {
				continue
			}

			sets = append(sets, set)
		}
	}

	if len(sets) == 0 && *setId != "" {
		set := getSpecificSet(appFlickrOAuth, *setId)
		sets = append(sets, set.Set)
	}

	if *setId == "" || *onlyPhotosNotInSet {
		// Handle photos not in a set if we haven't targeted
		// a specific set
		noSet := new(Photoset)
		noSet.Id = ""
		noSet.Title = "NO-SET"
		sets = append(sets, *noSet)
	}

	return sets
}

/**
 * Processes a single set
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   FlickrOAuth   Flickr OAuth config
 * @param   Photoset      The set to process
 * @return  void
**/

func processSingleSet(appFlickrOAuth FlickrOAuth, setToProcess Photoset) {

	// Create the directory for this set with the set's created
	// date as the prefix so the directories are ordered the same way
	// flickr orders the sets
	dir := ensureDirForSet(setToProcess)

	// Get all the photos for this set
	var flickrItems map[string]Photo
	if len(setToProcess.Id) > 0 {
		flickrItems = getPhotosForSet(appFlickrOAuth, setToProcess)
	} else {
		flickrItems = getPhotosNotInSet(appFlickrOAuth)
	}

	// Get all the files on the filesystem, if any exist
	existingFiles, _ := ioutil.ReadDir(dir)

	metadataFile := filepath.Join(dir, setMetadataFileName)
	var metadata SetMetadata

	// Read the existing metadata, or create a new struct if none is found,
	// so we can pick up where we left off
	if pathExists(metadataFile) {
		existingMetadata, _ := ioutil.ReadFile(metadataFile)
		json.Unmarshal(existingMetadata, &metadata)
	} else {
		metadata = SetMetadata{Photos: []MediaMetadata{}, SetId: setToProcess.Id}
	}

	if *auditOnly == true {

		auditSet(existingFiles, &metadata, flickrItems, setToProcess, metadataFile, dir)
		return
	}

	if *forceProcessing != true {
		// Skip sets that already have all their files downloaded
		if len(existingFiles) == (len(flickrItems)+1) && len(flickrItems) == len(metadata.Photos) {
			logMessage(fmt.Sprintf("Skipping set: `%v'. Found %v existing files.", setToProcess.Title, strconv.Itoa(len(existingFiles))), false)
			return
		}

		formatString := "Processing set: `%v'. Found %v existing files on disk, %v files in metadata, and %v files on Flickr."
		logMessage(fmt.Sprintf(formatString, setToProcess.Title, strconv.Itoa(len(existingFiles)), strconv.Itoa(len(metadata.Photos)), strconv.Itoa(len(flickrItems))), false)
	} else {
		logMessage(fmt.Sprintf("Force processing set: `%v'", setToProcess.Title), false)
	}

	var fullPath string
	var fileName string
	var sourceUrl string
	var mediaType string
	for _, media := range flickrItems {

		// Get the photo and video url (if one exists)
		photoUrl, videoUrl := getOriginalSizeUrl(appFlickrOAuth, media)

		if videoUrl != "" {

			fileName = media.Id + ".mov"
			sourceUrl = videoUrl
			mediaType = "video"

		} else if photoUrl != "" {

			fileName = getFileNameFromUrl(photoUrl)
			sourceUrl = photoUrl
			mediaType = "photo"

		} else {

			logMessage(fmt.Sprintf("Could not get original size for media: `%v' (%v). Skipping media for now.", media.Title, media.Id), true)
			continue
		}

		fullPath = filepath.Join(dir, fileName)

		// Skip files that exist
		if pathExists(fullPath) {
			logMessage(fmt.Sprintf("Media existed at %v. Skipping.", fullPath), false)
			metadata.AddOrUpdate(MediaMetadata{PhotoId: media.Id, Title: media.Title, Filename: fileName}, metadataFile)
			continue
		}

		// Save media to disk
		saveUrlToFile(func() string { return sourceUrl }, fullPath)

		// Add the photos metadata to the list and write the metadata file out
		metadata.AddOrUpdate(MediaMetadata{PhotoId: media.Id, Title: media.Title, Filename: fileName}, metadataFile)
		logMessage(fmt.Sprintf("Saved %v `%v' to %v.", mediaType, media.Title, fullPath), false)
	}

	// Look through all the files in the metadata and find the ones that no longer exist in
	// Flickr. Note them and then loop over those to delete them from the filesystem and
	// remove them from the metadata
	filesToRemove := map[string]string{}
	for _, pm := range metadata.Photos {
		if _, ok := flickrItems[pm.PhotoId]; !ok {
			fullPath = filepath.Join(dir, pm.Filename)
			filesToRemove[fullPath] = pm.PhotoId
		}
	}

	for photoFilePath, mediaId := range filesToRemove {

		logMessage(fmt.Sprintf("Deleting media Id `%v' at `%v'", mediaId, photoFilePath), true)
		deleteFile(photoFilePath)
		metadata.RemoveItemById(mediaId, metadataFile)
	}

}

/**
 * Ensures the directory for a set exists on disk
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   Photoset   The set to consider
 * @return  string     The resulting directory path
**/

func ensureDirForSet(set Photoset) string {

	var dir string
	if len(set.Id) > 0 {
		t := time.Unix(int64(set.DateCreated), 0)
		format := "20060102"
		dir = filepath.Join(*rootDirectory, fmt.Sprintf("%v %v", t.Format(format), set.CleanTitle()))
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			panic(err)
		}
	} else {
		dir = filepath.Join(*rootDirectory, "NO-SET")
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			panic(err)
		}
	}

	return dir
}
