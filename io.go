package main

import (
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strings"
)

var perms os.FileMode = 0700

/**
 * Determines if a file exists on disk
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   string    The full path to the file to test
 * @return  bool
**/

func pathExists(fullPath string) bool {

	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		return true
	}

	return false

}

/**
 * Given a UrlFunc and file path, save the contents of the url to the file location
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   UrlFunc    The function to generate the url
 * @param   string     The full path to save the contents to
 * @return  void
**/

func saveUrlToFile(urlGenerator UrlFunc, fullPath string) {

	var err error
	var body []byte

	body, err = makeGetRequest(urlGenerator)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(fullPath, body, 0644)
}

/**
 * From a flickr url, get the filename piec
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   string   The flickr url
 * @return  string   The filename extracted from the url
**/

func getFileNameFromUrl(url string) string {

	index := strings.LastIndex(url, "/")
	if index == -1 {
		return ""
	}
	return url[index+1:]
}

/**
 * Deletes a file from the filesystem
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   string   The full path to delete
 * @return  void
**/

func deleteFile(fullPath string) {

	os.Remove(fullPath)
}

func getUserFilePath(fileName string) string {

	dir := ensureUserHomeDir()
	filePath := path.Join(dir, fileName)
	return filePath
}

func ensureUserHomeDir() string {

	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	dir := path.Join(usr.HomeDir, ".fsync")
	if !pathExists(dir) {
		os.Mkdir(dir, perms)
	}

	return dir
}
