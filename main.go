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
var generateApiSignature = flag.Bool("genApiSig", false, "Print the api signature for a given request url. Useful when debugging an invalid signature response from Flickr. Paste the 'debug_sbs' value they send back.")
var debugSbs = flag.String("debug_sbs", "", "The debug_sbs return parameter from Flickr.")
var Flogger *log.Logger

var setMetadataFileName = "metadata.json"

func main() {

	flag.Parse()

	Flogger = createLogger()

	secrets := loadOAuthSecrets()
	if !secrets.isValid() {
		logMessage("Your OAuth secrets file doesn't exist or is invalid. See the log file for more details.", true)
		return
	}

	if *generateApiSignature == true {
		signature := getApiSignature(*debugSbs)
		if len(signature) > 0 {
			fmt.Println(signature)
		}
		return
	}

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
