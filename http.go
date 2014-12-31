package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"strconv"
	"strings"
)

var lastRequestTime time.Time

type UrlFunc func() string

/**
 * Makes a Http GET request.
 *
 * Ensures that we make requests at no faster than 1 per second to
 * avoid Flickr's api TOS violations.
 *
 * Also check for an "invalid signature" error from Flickr
 * which seems to happen sometimes. If we run into this
 * error, we retry up to 10 times, generating a new url
 * on each retry, which is why we pass in a UrlFunc to the 
 * function, rather than just a string url.
 *
 * Its possible I'm calculating the api signature incorrectly
 * sometimes, but I can't seem to find the problem.
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   UrlFunc         The function to generate a url for retrying a failed request
 * @return  []byte, error   The byte array of the response and any error
**/
 
func makeGetRequest(generateUrlFunction UrlFunc) ([]byte, error) {

	currentTime := time.Now()
	if ! lastRequestTime.IsZero() {

		// Sleep until we make sure we don't make requests faster than 1/sec
		nano := currentTime.Sub(lastRequestTime)
		milli := nano * 1000000

		if milli < 1000 && milli > 0 {
			logMessage(fmt.Sprintf("Sleeping for %v milliseconds before making another request.", milli), false)
			time.Sleep(milli * time.Millisecond)
		}
	}

	lastRequestTime = currentTime
	retryCount := 0

	for {

		var resp *http.Response
		var err error

		url := generateUrlFunction()
		resp, err = http.Get(url)
		if err != nil {
			return []byte{}, err
		}

		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return []byte{}, err
		}

		if strings.Contains(string(body), "oauth_problem=signature_invalid") && retryCount < 10 {
			retryCount++
			logMessage("Sleeping and retrying request, retry #" + strconv.Itoa(retryCount), false)
			time.Sleep(1 * time.Second)
		} else {
			return body, nil
		}
	}
}
