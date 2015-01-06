package main

import (
	"fmt"
	"os"
	"path/filepath"
)

/**
 * Finds duplicate media files and lists them to the console
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @return  void
**/

func findDupes() {

	duplicates := map[string][]string{}
	visitor := func(path string, f os.FileInfo, err error) error {

		if f.IsDir() {
			return nil
		}

		if f.Name() == "metadata.json" {
			return nil
		}

		if _, ok := duplicates[f.Name()]; !ok {
			duplicates[f.Name()] = []string{}
		}

		duplicates[f.Name()] = append(duplicates[f.Name()], path)

		return nil
	}

	filepath.Walk(*rootDirectory, visitor)

	var totalDupes = 0
	for fileName, paths := range duplicates {

		if len(paths) < 2 {
			continue
		}

		totalDupes += len(paths) - 1
		logMessage(fmt.Sprintf("File `%v' was found %v times.", fileName, len(paths)), false)
		for _, path := range paths {
			logMessage(path, true)
		}
	}

	photoCount, movieCount := countMediaFiles()
	realMediaCount := (photoCount + movieCount) - totalDupes

	logMessage(fmt.Sprintf("Total dupes: %v. Real count of media files: %v", totalDupes, realMediaCount), true)
}
