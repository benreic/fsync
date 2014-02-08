package main

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
)

var apiBaseUrl = "http://api.flickr.com/services/rest"
var setsCacheFile = "cache/sets.xml"

type PhotosetsResponse struct {
	XMLName      xml.Name `xml:"rsp"`
	SetContainer Photosets
}

type Photosets struct {
	XMLName xml.Name   `xml:"photosets"`
	Total   string     `xml:"total,attr"`
	Sets    []Photoset `xml:"photoset"`
}

type Photoset struct {
	XMLName     xml.Name `xml:"photoset"`
	Id          string   `xml:"id,attr"`
	DateCreated int      `xml:"date_create,attr"`
	Photos      int      `xml:"photos,attr"`
	Videos      int      `xml:"videos,attr"`
	Title       string   `xml:"title"`
}

type PhotosResponse struct {
	XMLName xml.Name `xml:"rsp"`
	Set     PhotosPhotoset
}

type PhotosPhotoset struct {
	XMLName xml.Name `xml:"photoset"`
	Id      string   `xml:"id,attr"`
	Photos  []Photo  `xml:"photo"`
}

type Photo struct {
	XMLName xml.Name `xml:"photo"`
	Id      string   `xml:"id,attr"`
	Title   string   `xml:"title,attr"`
}

type PhotoSizeResponse struct {
	XMLName        xml.Name           `xml:"rsp"`
	SizesContainer PhotoSizeContainer `xml:"sizes"`
}

type PhotoSizeContainer struct {
	XMLName xml.Name    `xml:"sizes"`
	Sizes   []PhotoSize `xml:"size"`
}

type PhotoSize struct {
	XMLName xml.Name `xml:"size"`
	Label   string   `xml:"label,attr"`
	Url     string   `xml:"source,attr"`
}

type ByDateCreated []Photoset

func (a ByDateCreated) Len() int           { return len(a) }
func (a ByDateCreated) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByDateCreated) Less(i, j int) bool { return a[i].DateCreated < a[j].DateCreated }

func getSets(flickrOAuth FlickrOAuth) PhotosetsResponse {

	var body []byte
	var err error
	if _, err := os.Stat(setsCacheFile); os.IsNotExist(err) {

		getSetsUrl := generateGetSetsUrl(flickrOAuth)

		body, err = makeGetRequest(getSetsUrl)
		if err != nil {
			panic(err)
		}

		ioutil.WriteFile(setsCacheFile, body, 0644)

	} else {
		body, err = ioutil.ReadFile(setsCacheFile)
		if err != nil {
			panic(err)
		}
	}

	sets := PhotosetsResponse{}
	err = xml.Unmarshal(body, &sets)
	if err != nil {
		logMessage("Could not unmarshal body. Check logs for body detail.", true)
		logMessage(string(body), false)
		panic(err)
	}

	sort.Sort(ByDateCreated(sets.SetContainer.Sets))

	return sets
}

func getPhotosForSet(flickrOAuth FlickrOAuth, set Photoset) map[string]Photo {

	var err error
	var body []byte
	photos := map[string]Photo{}
	currentPage := 1

	for {
		extras := map[string]string{"photoset_id": set.Id, "per_page": "500", "page": strconv.Itoa(currentPage)}
		requestUrl := generateOAuthUrl(apiBaseUrl, "flickr.photosets.getPhotos", flickrOAuth, &extras)

		body, err = makeGetRequest(requestUrl)
		if err != nil {
			panic(err)
		}

		response := PhotosResponse{}
		err = xml.Unmarshal(body, &response)
		if err != nil {
			logMessage("Could not unmarshal body. Check logs for body detail.", true)
			logMessage(string(body), false)
			panic(err)
		}

		for _, v := range response.Set.Photos {
			photos[v.Id] = v
		}

		if len(photos) == (set.Photos + set.Videos) {
			break
		}

		currentPage++
	}

	return photos
}

func getOriginalSizeUrl(flickrOauth FlickrOAuth, photo Photo) string {

	extras := map[string]string{"photo_id": photo.Id}
	requestUrl := generateOAuthUrl(apiBaseUrl, "flickr.photos.getSizes", flickrOauth, &extras)

	var err error
	var body []byte

	body, err = makeGetRequest(requestUrl)
	if err != nil {
		panic(err)
	}

	response := PhotoSizeResponse{}
	err = xml.Unmarshal(body, &response)
	if err != nil {
		logMessage("Could not unmarshal body. Check logs for body detail.", true)
		logMessage(string(body), false)
		panic(err)
	}

	for _, v := range response.SizesContainer.Sizes {
		if v.Label == "Original" {
			return v.Url
		}
	}

	return ""
}

func savePhotoToFile(flickrOauth FlickrOAuth, url string, fullPath string) {

	var err error
	var body []byte

	body, err = makeGetRequest(url)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(fullPath, body, 0644)
}

func generateGetSetsUrl(flickrOauth FlickrOAuth) string {

	requestUrl := generateOAuthUrl(apiBaseUrl, "flickr.photosets.getList", flickrOauth, nil)
	return requestUrl
}
