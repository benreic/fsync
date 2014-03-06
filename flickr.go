package main

import (
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
)

var apiBaseUrl = "http://api.flickr.com/services/rest"

type FlickrErrorResponse struct {
	XMLName xml.Name `xml:"rsp"`
	Error   FlickrError
}

type FlickrError struct {
	XMLName xml.Name `xml:"err"`
	Code    string   `xml:"code,attr"`
	Message string   `xml:"msg,attr"`
}

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

	requestUrl := generateGetSetsUrl(flickrOAuth)

	body, err := makeGetRequest(requestUrl)
	if err != nil {
		panic(err)
	}

	sets := PhotosetsResponse{}
	err = xml.Unmarshal(body, &sets)
	if err != nil {
		logMessage(fmt.Sprintf("Could not unmarshal body for `%v'. Check logs for body detail.", requestUrl), true)
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

			// We couldn't unmarshal the response as photos, but it might be the case
			// that we just ran out of photos, i.e. the set has a multiple of 500 photos in it
			// Lets try to unmarshal the response as an error, and if it is, error code "1" means
			// we're good and we can take what we've got and roll on.
			errorResponse := FlickrErrorResponse{}
			err = xml.Unmarshal(body, &errorResponse)
			if err != nil {
				logMessage(fmt.Sprintf("Could not unmarshal body for `%v' Tried PhotosResponse and then FlickrErrorResponse. Check logs for body detail.", requestUrl), true)
				logMessage(string(body), false)
				panic(err)
			}

			// The "good" error code
			if errorResponse.Error.Code == "1" {
				break
			}

			logMessage(fmt.Sprintf("An error occurred while getting photos for the set. Check the body in the logs. Url: %v", requestUrl), false)
			logMessage(string(body), false)
		}

		for _, v := range response.Set.Photos {
			photos[v.Id] = v
		}

		// If we didn't get 500 photos, then we're done.
		// There are no more photos to get.
		if len(response.Set.Photos) < 500 {
			break
		}

		currentPage++
	}

	return photos
}

func getOriginalSizeUrl(flickrOauth FlickrOAuth, photo Photo) (string, string) {

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
		logMessage(fmt.Sprintf("Could not unmarshal body for `%v'. Check logs for body detail.", requestUrl), true)
		logMessage(string(body), false)
		return "", ""
	}

	photoUrl := ""
	videoUrl := ""
	for _, v := range response.SizesContainer.Sizes {
		if v.Label == "Original" {
			photoUrl = v.Url
		}

		if v.Label == "Video Original" {
			videoUrl = v.Url
		}
	}

	return photoUrl, videoUrl
}

func generateGetSetsUrl(flickrOauth FlickrOAuth) string {

	requestUrl := generateOAuthUrl(apiBaseUrl, "flickr.photosets.getList", flickrOauth, nil)
	return requestUrl
}
