package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

var lastRequestTime = time.Now()

func makeGetRequest(url string) ([]byte, error) {

	currentTime := time.Now()

	nano := currentTime.Sub(lastRequestTime)
	milli := nano * 1000000

	if milli < 1000 {
		logMessage(fmt.Sprintf("Sleeping for %v milliseconds before making another request.", milli), false)
		time.Sleep(milli * time.Millisecond)
	}

	lastRequestTime = currentTime

	var resp *http.Response
	var err error

	resp, err = http.Get(url)
	if err != nil {
		// Wait 1 second and try again
		time.Sleep(1 * time.Second)
		resp, err = http.Get(url)
		if err != nil {
			return []byte{}, err
		}
	}

	defer resp.Body.Close()

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return body, nil
}
