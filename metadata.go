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

func (sm SetMetadata) Save(metadataFile string) {

	metadataBytes, _ := json.Marshal(sm)
	ioutil.WriteFile(metadataFile, metadataBytes, 0755)
}

func (sm SetMetadata) RemoveItemById (id string, metadataFile string) {

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

func (sm SetMetadata) RemoveItemByFilename (fileName string, metadataFile string) {

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
