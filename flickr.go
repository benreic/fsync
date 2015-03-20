package main

import (
	"encoding/xml"
	"sort"
	"strconv"
	"strings"
)

var apiBaseUrl = "https://api.flickr.com/services/rest"
var getPhotosInSetName = "flickr.photosets.getPhotos"
var getPhotosNotInSetName = "flickr.photos.getNotInSet"

type FlickrErrorResponse struct {
	XMLName xml.Name `xml:"rsp"`
	Error   FlickrError
}

type FlickrError struct {
	XMLName xml.Name `xml:"err"`
	Code    string   `xml:"code,attr"`
	Message string   `xml:"msg,attr"`
}

// Get a single set
type SinglePhotosetResponse struct {
	XMLName xml.Name `xml:"rsp"`
	Set     Photoset `xml:"photoset"`
}

// Get list of sets
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

func (ps Photoset) CleanTitle() string {

	invalidChars := []string{"\\", "/", ":", ">", "<", "?", "\"", "|", "*"}

	title := ps.Title
	for _, char := range invalidChars {
		title = strings.Replace(title, char, "", -1)
	}

	return title
}

// Get photos not in a set
type PhotosNotInSetResponse struct {
	XMLName xml.Name `xml:"rsp"`
	Photos  []Photo  `xml:"photos>photo"`
}

// Get list of photos from a set
type PhotosResponse struct {
	XMLName xml.Name `xml:"rsp"`
	Set     PhotosPhotoset
}

type PhotosPhotoset struct {
	XMLName xml.Name `xml:"photoset"`
	Id      string   `xml:"id,attr"`
	Photos  []Photo  `xml:"photo"`
}

// Used by both in-set and not-in-set photos responses
type Photo struct {
	XMLName     xml.Name `xml:"photo"`
	Id          string   `xml:"id,attr"`
	Title       string   `xml:"title,attr"`
	OriginalUrl string   `xml:"url_o,attr"`
	Media       string   `xml:"media,attr"`
}

// Get sizes of photos
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

/**
 * Gets all sets for the user
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   FlickrOAuth    The flickr oauth setup
 * @return  PhotosetsResponse
**/

func getSets(flickrOAuth FlickrOAuth) PhotosetsResponse {

	body, err := makeGetRequest(func() string { return generateGetSetsUrl(flickrOAuth) })
	if err != nil {
		panic(err)
	}

	sets := PhotosetsResponse{}
	err = xml.Unmarshal(body, &sets)
	if err != nil {
		logMessage("Could not unmarshal body, check logs for body detail.", true)
		logMessage(string(body), false)
		panic(err)
	}

	sort.Sort(ByDateCreated(sets.SetContainer.Sets))

	return sets
}

/**
 * Gets a set by set Id
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   FlickrOAuth    The flickr oauth setup
 * @return  PhotosetsResponse
**/

func getSpecificSet(flickrOAuth FlickrOAuth, setId string) SinglePhotosetResponse {

	extras := map[string]string{"photoset_id": setId}
	body, err := makeGetRequest(func() string { return generateOAuthUrl(apiBaseUrl, "flickr.photosets.getInfo", flickrOAuth, extras) })
	if err != nil {
		panic(err)
	}

	set := SinglePhotosetResponse{}
	err = xml.Unmarshal(body, &set)
	if err != nil {
		logMessage("Could not unmarshal body, check logs for body detail.", true)
		logMessage(string(body), false)
		panic(err)
	}

	return set
}

/**
 * Gets all the media for a given set
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   FlickrOAuth        The flickr oauth setup
 * @return  map[string]Photo   The list of media files, indexed by Flickr Id
**/

func getPhotosForSet(flickrOAuth FlickrOAuth, set Photoset) map[string]Photo {

	return getAllPhotos(flickrOAuth, getPhotosInSetName, set.Id)
}

/**
 * Gets all media that is not included in a set
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   FlickrOAuth        The flickr oauth setup
 * @return  map[string]Photo   The list of media files indexed by Flickr Id
**/

func getPhotosNotInSet(flickrOAuth FlickrOAuth) map[string]Photo {

	return getAllPhotos(flickrOAuth, getPhotosNotInSetName, "")
}

/**
 * Actually does the work to get the media files
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   FlickrOAuth        The flickr oauth setup
 * @param   string             Which flickr api we're using (with set or w/o)
 * @param   string             The set id of media files we're getting.
 * @return  map[string]Photo   The list of media files indexed by Flickr Id
**/

func getAllPhotos(flickrOAuth FlickrOAuth, apiName string, setId string) map[string]Photo {

	var err error
	var body []byte
	photos := map[string]Photo{}
	currentPage := 1
	pageSize := 500

	for {

		extras := map[string]string{"page": strconv.Itoa(currentPage)}
		extras["per_page"] = strconv.Itoa(pageSize)
		extras["extras"] = "media,url_o"
		if len(setId) > 0 {
			extras["photoset_id"] = setId
		}

		body, err = makeGetRequest(func() string { return generateOAuthUrl(apiBaseUrl, apiName, flickrOAuth, extras) })
		if err != nil {
			panic(err)
		}

		responsePhotos := []Photo{}
		var err error
		if apiName == getPhotosNotInSetName {
			response := PhotosNotInSetResponse{}
			err = xml.Unmarshal(body, &response)
			if err == nil {
				responsePhotos = response.Photos
			}
		} else {
			response := PhotosResponse{}
			err = xml.Unmarshal(body, &response)
			if err == nil {
				responsePhotos = response.Set.Photos
			}
		}

		if err != nil {

			// We couldn't unmarshal the response as photos, but it might be the case
			// that we just ran out of photos, i.e. the set has a multiple of 500 photos in it
			// Lets try to unmarshal the response as an error, and if it is, error code "1" means
			// we're good and we can take what we've got and roll on.
			errorResponse := FlickrErrorResponse{}
			err = xml.Unmarshal(body, &errorResponse)
			if err != nil {

				logMessage("Could not unmarshal body, check logs for body detail.", true)
				logMessage(string(body), false)
				panic(err)
			}

			// The "good" error code
			if errorResponse.Error.Code == "1" {
				break
			}

			logMessage("An error occurred while getting photos for the set. Check the body in the logs.", false)
			logMessage(string(body), false)
		}

		for _, v := range responsePhotos {
			photos[v.Id] = v
		}

		// If we didn't get 500 photos, then we're done.
		// There are no more photos to get.
		if len(responsePhotos) < pageSize {
			break
		}

		currentPage++
	}

	return photos
}

/**
 * Gets the original size url for a given Flickr media
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   FlickrOAuth        The flickr oauth setup
 * @param   Photo              The flickr media to consider
 * @return  string,string      A photo url and a video url
**/

func getOriginalSizeUrl(flickrOauth FlickrOAuth, photo Photo) (string, string) {

	if photo.Media == "photo" {
		return photo.OriginalUrl, ""
	}

	extras := map[string]string{"photo_id": photo.Id}

	var err error
	var body []byte

	body, err = makeGetRequest(func() string { return generateOAuthUrl(apiBaseUrl, "flickr.photos.getSizes", flickrOauth, extras) })
	if err != nil {
		panic(err)
	}

	response := PhotoSizeResponse{}
	err = xml.Unmarshal(body, &response)
	if err != nil {
		logMessage("Could not unmarshal body, check logs for body detail.", true)
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

/**
 * Helper function to create the url used to get the list of sets
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   FlickrOAuth        The flickr oauth setup
 * @return  string             The url to use
**/

func generateGetSetsUrl(flickrOauth FlickrOAuth) string {

	requestUrl := generateOAuthUrl(apiBaseUrl, "flickr.photosets.getList", flickrOauth, nil)
	return requestUrl
}
