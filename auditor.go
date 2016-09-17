package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

/**
 *
 * Loop through the photos in the set. See if each media exists in the metadata. Keep track of photos
 * that don't exist in the metadata, these need to be downloaded.
 * Loop through the media in the metadata. Any that don't exist in the set should be deleted and removed from the metadata.
 * Loop through the file and make sure they are all in the metadata.
 *
 **/

func auditSet(existingFiles []os.FileInfo, metadata *SetMetadata, photos map[string]Photo, set Photoset, metadataFile string, setDir string) {

	logMessage(fmt.Sprintf("Auditing set: `%v'", set.Title), true)

	// Convert the metadata into a map for ease of use
	photoIdMap := map[string]MediaMetadata{}
	fileNameMap := map[string]MediaMetadata{}

	for _, pm := range metadata.Photos {
		photoIdMap[pm.PhotoId] = pm
		fileNameMap[pm.Filename] = pm
	}

	// Find photos that don't exist on disk and need to be downloaded
	for mediaId, photo := range photos {

		_, valueExists := photoIdMap[mediaId]
		if valueExists == false {

			doLog := true
			for _, fi := range existingFiles {
				if strings.Index(fi.Name(), mediaId) == 0 {
					logMessage(fmt.Sprintf("Media Id `%v' (%v) does not exist in the metadata, but the media appears to exist on disk with file name `%v'. It needs to be added to the metadata.", mediaId, photo.Title, fi.Name()), true)
					doLog = false
					break
				}
			}

			if doLog {
				logMessage(fmt.Sprintf("Media Id `%v' (%v) does not exist in the metadata. It needs to be downloaded and added to the metadata.", mediaId, photo.Title), true)
			}
		}
	}

	// Find photos that exist on disk, but not in Flickr, they need to be deleted.
	for photoId, pm := range photoIdMap {

		if _, ok := photos[photoId]; !ok {
			logMessage(fmt.Sprintf("Media Id `%v' (%v) does not exist in Flickr and needs to be deleted.", photoId, pm.Title), true)
		}
	}

	// Find photos on disk that are not in the metadata
	for _, fi := range existingFiles {
		if fi.Name() == setMetadataFileName {
			continue
		}
		_, valueExists := fileNameMap[fi.Name()]
		if valueExists == false {
			logMessage(fmt.Sprintf("Media exists on disk, but not in metadata. This is a bug.: `%v'.", fi.Name()), true)
		}
	}

	// Find photos in metadata that are not on disk
	for fileName, _ := range fileNameMap {
		if fileName == setMetadataFileName {
			continue
		}

		// make the full file path from the filename
		fullFileName := filepath.Join(setDir, fileName)
		if !pathExists(fullFileName) {
			logMessage(fmt.Sprintf("File exists in metadata, but not on disk. The file was either deleted or never saved correctly. This is a bug.: `%v'.", fullFileName), true)
		}
	}
}
