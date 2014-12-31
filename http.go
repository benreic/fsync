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

func makeGetRequest(generateUrlFunction UrlFunc) ([]byte, error) {

	currentTime := time.Now()
	if ! lastRequestTime.IsZero() {

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
