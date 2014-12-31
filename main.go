package main

import (
	"flag"
	"fmt"
	"log"
)

var appFlickrOAuth = new(FlickrOAuth)
var rootDirectory = flag.String("dir", "", "The base directory where your sets/photos will be downloaded.")
var setId = flag.String("setId", "", "Only process a single set; applies to audit and actual processing")
var forceProcessing = flag.Bool("force", false, "Force processing of each set; don't skip sets even if file counts match")
var auditOnly = flag.Bool("audit", false, "Compares existing media with the media on Flickr and displays the differences")
var countOnly = flag.Bool("count", false, "Recursively counts all media files in the specified directory")
var findDuplicates = flag.Bool("dupes", false, "Find and print media files that exist in multiple sets.")
var onlyPhotosNotInSet = flag.Bool("onlyNonSet", false, "Skip all sets and only process media that are not in a set")
var Flogger *log.Logger

var setMetadataFileName = "metadata.json"

func main() {
 
	flag.Parse()

	Flogger = createLogger()

	if *rootDirectory == "" {
		fmt.Println("You must specify a root directory using -dir")
		return
	}

	if *countOnly == true {
		countFiles()
		return
	}

	if *findDuplicates == true {
		findDupes()
		return
	}

	processSets()
}


