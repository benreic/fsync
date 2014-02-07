package main

import (
	"io/ioutil"
	"net/http"
	"time"
)

func makeGetRequest(url string) ([]byte, error) {

	var resp *http.Response
	var err error

	resp, err = http.Get(url)
	if err != nil {
		// Wait 500 milliseconds and try again
		time.Sleep(500 * time.Millisecond)
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
