package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type SetMetadata struct {
	SetId  string
	Photos []MediaMetadata
}

// Media metadata struct
type MediaMetadata struct {
	PhotoId  string
	Title    string
	Filename string
}


/**
 * Saves the metadata to its json file
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   string   The metadata json filename
 * @return  void
**/
 
func (sm SetMetadata) Save(metadataFile string) {

	metadataBytes, _ := json.Marshal(sm)
	ioutil.WriteFile(metadataFile, metadataBytes, 0755)
}


/**
 * Removes a media item from the metadata and saves the metadata to disk
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   string   The media id to remove
 * @param   string   The metadata filename
 * @return  void
**/
 
func (sm *SetMetadata) RemoveItemById (id string, metadataFile string) {

	var newListOfMedia = []MediaMetadata{ }
	for _, photo := range sm.Photos { 
		if photo.PhotoId != id {
			newListOfMedia = append(newListOfMedia, photo)
		} else {
			logMessage(fmt.Sprintf("Removing Id `%v' from the metadata.", id), true)
		}
	}

	sm.Photos = newListOfMedia
	sm.Save(metadataFile)
}


/**
 * Removes a media item by filename instead of id, and saves the metadata to disk
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   string   The filename to remove
 * @param   string   The metadata json filename
 * @return  void
**/
 
func (sm *SetMetadata) RemoveItemByFilename (fileName string, metadataFile string) {

	var newListOfMedia = []MediaMetadata{ }
	for _, photo := range sm.Photos { 
		if photo.Filename != fileName {
			newListOfMedia = append(newListOfMedia, photo)
		} else {
			logMessage(fmt.Sprintf("Removing filename `%v' from the metadata.", fileName), true)
		}
	}

	sm.Photos = newListOfMedia
	sm.Save(metadataFile)
}


/**
 * Adds or updates a media item to the metadata and saves the metadata to disk.
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   MediaMetadata    The item to add or update
 * @param   string           The json filename 
 * @return  void
**/
 
func (sm *SetMetadata) AddOrUpdate (p MediaMetadata, metadataFile string) {

	// See if there is an existing entry for this photo
	// update the metadata if there is
	var foundPhoto = false
	for index, photo := range sm.Photos {
		if photo.PhotoId == p.PhotoId {
			sm.Photos[index].Title = p.Title
			sm.Photos[index].Filename = p.Filename
			foundPhoto = true
			logMessage("Updating existing entry in metadata.", false)
			break
		}
	}

	// Didn't find an existing one, so add to the metadata
	if ! foundPhoto {
		slice := append(sm.Photos, p)
		sm.Photos = slice
	}

	sm.Save(metadataFile)
}


