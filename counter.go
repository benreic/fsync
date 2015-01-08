package main

import (
	"fmt"
	"os"
	"path/filepath"
)

/**
 * Echos the number of media files to the console
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @return  void
**/

func countFiles() {

	photoCount, movieCount := countMediaFiles()
	logMessage(fmt.Sprintf("Found %v media files, including duplicates (photos can be part of more than one album). (%v photos, %v movies)", (photoCount+movieCount), photoCount, movieCount), true)
}

/**
 * Counts the number of media files
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @return  int,int   The number of photos and the number of videos
**/

func countMediaFiles() (int, int) {

	var photoCount = 0
	var movieCount = 0
	visitor := func(path string, f os.FileInfo, err error) error {

		if !f.IsDir() {
			return nil
		}

		matches, _ := filepath.Glob(filepath.Join(path, "*.jpg"))
		if matches != nil {
			photoCount += len(matches)
		}

		matches, _ = filepath.Glob(filepath.Join(path, "*.gif"))
		if matches != nil {
			photoCount += len(matches)
		}

		matches, _ = filepath.Glob(filepath.Join(path, "*.png"))
		if matches != nil {
			photoCount += len(matches)
		}

		matches, _ = filepath.Glob(filepath.Join(path, "*.mov"))
		if matches != nil {
			movieCount += len(matches)
		}

		return nil
	}

	filepath.Walk(*rootDirectory, visitor)
	return photoCount, movieCount
}
