package main

import (
	"fmt"
	"strings"
	"time"
	"os/exec"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"net/url"
	"encoding/json"
	"os"
)

var oauth_request_token_url = "http://www.flickr.com/services/oauth/request_token"
var oauth_exchange_token_url = "http://www.flickr.com/services/oauth/access_token"
var oauth_nonce = ""
var cacheFile = "cache/oauth.json"
var oauthSecretsFile = "cache/oauth-secrets.json"

type FlickrOAuth struct {
	FullName string
	OAuthToken string
	OAuthTokenSecret string
	UserNSID string
	Username string
}

type OAuthSecrets struct {
	ConsumerKey string
	Secret string
	MinitokenUrl string
}

// Loads the secret oauth data for this app from a json file. This file
// should not get committed to source control. (The entire cache directory
// should be ignored, really.)
func loadOAuthSecrets() OAuthSecrets {

	s := new(OAuthSecrets)
	if _, err := os.Stat(oauthSecretsFile); ! os.IsNotExist(err) {
		fileContents, _ := ioutil.ReadFile(oauthSecretsFile)
		if len(fileContents) > 0 {
			json.Unmarshal(fileContents, &s)
		} else {
			panic("oauth-secrets.json file was empty.")
		}
	} else {
		panic("oauth-secrets.json didn't exist.")
	}

	return *s
}

// Checks for cached oauth credentials so we don't need to 
// go through the oauth process again.
func checkForExistingOAuthCredentials() FlickrOAuth {

	var oauth = new(FlickrOAuth)
	if _, err := os.Stat(cacheFile); ! os.IsNotExist(err) {
		fileContents, _ := ioutil.ReadFile(cacheFile)
		if len(fileContents) > 0 {
			json.Unmarshal(fileContents, &oauth)
		}
	}

	return *oauth
}

// Does the oauth handshaking with flickr and the user
func doOAuthSetup() FlickrOAuth {

	oauthResult := FlickrOAuth { "", "", "", "", "" }

	// Get the request token url
	var oauth_request_token_request = generateRequestTokenUrl()

	// Get the response from the request token url
	// and check for errors
	resp, err := http.Get(oauth_request_token_request)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// Parse the oauth token and oauth token secret values
	// from the body response.
	oauth_token := ""
	oauth_token_secret := ""
	parts := strings.Split(string(body), "&")
	for _, element := range parts {
		piece := strings.Split(element, "=")
		if piece[0] == "oauth_callback_confirmed" {
			if piece[1] != "true" {
				fmt.Println("Bad oauth result:\n" + string(body))
				return oauthResult
			}
		}
		if piece[0] == "oauth_token" {
			oauth_token = piece[1]
		}
		if piece[0] == "oauth_token_secret" {
			oauth_token_secret = piece[1]
		}
	}

	// Bail if we don't have a token or token secret
	if oauth_token == "" {
		panic("No oauth token found")
	}

	if oauth_token_secret == "" {
		panic("No oauth token secret found")
	}

	// Send the user to flickr to authorize us
	exec.Command("open", "http://www.flickr.com/services/oauth/authorize?perms=read&oauth_token="+oauth_token).Start()

	// Have them enter the 9 digit code from flickr
	fmt.Println("Authorize the app on flickr's site and enter the nine digit code here and press 'Return':")
	var userToken = ""
	_, err = fmt.Scanln(&userToken)

	// Generate the exchange token url
	exchangeForRealTokenUrl := generateExchangeUrl(userToken, oauth_token, oauth_token_secret)

	// Get the response and check for errors
	resp, err = http.Get(exchangeForRealTokenUrl)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// Parse the result for the oauth token
	parts = strings.Split(string(body), "&")
	for _, element := range parts {

		piece := strings.Split(element, "=")

		if piece[0] == "fullname" {
			oauthResult.FullName = piece[1]
		}

		if piece[0] == "oauth_token" {
			oauthResult.OAuthToken = piece[1]
		}

		if piece[0] == "oauth_token_secret" {
			oauthResult.OAuthTokenSecret = piece[1]
		}

		if piece[0] == "user_nsid" {
			oauthResult.UserNSID = piece[1]
		}

		if piece[0] == "username" {
			oauthResult.Username = piece[1]
		}
	}

	b, err := json.Marshal(oauthResult)

	err = ioutil.WriteFile(cacheFile, b, 0644)
	if err != nil { panic(err) }
	return oauthResult
}

// Generates an oauth url for flickr based on the method the user wants and any extra params
func generateOAuthUrl(baseUrl string, method string, auth FlickrOAuth, extraParams *map[string]string) string {

	secrets := loadOAuthSecrets()

	params := make(map[string]string)
	params["format"] = "rest"
	params["method"] = method
	params["oauth_consumer_key"] = secrets.ConsumerKey
	params["oauth_nonce"] = generateNonce()
	params["oauth_signature_method"] = "HMAC-SHA1"
	params["oauth_timestamp"] = fmt.Sprintf("%v", time.Now().Unix())
	params["oauth_token"] = auth.OAuthToken
	params["oauth_version"] = "1.0"

	if extraParams != nil {

		for key, element := range *extraParams {
			params[key] = element
		}
	}


	apiSignature := createApiSignature(baseUrl, "GET", params, secrets.Secret, &auth.OAuthTokenSecret)
	params["oauth_signature"] = apiSignature

	requestUrl := baseUrl + "?"
	for key, element := range params {
		requestUrl += key + "=" + element + "&"
	}

	requestUrl = strings.TrimRight(requestUrl, "&")
	return requestUrl
}


// Generates the exchange token url, used during oauth handshaking
func generateExchangeUrl(userToken string, oauthToken string, tokenSecret string) string {

	secrets := loadOAuthSecrets()

	params := make(map[string]string)
	params["oauth_consumer_key"] = secrets.ConsumerKey
	params["oauth_nonce"] = generateNonce()
	params["oauth_signature_method"] = "HMAC-SHA1"
	params["oauth_timestamp"] = fmt.Sprintf("%v", time.Now().Unix())
	params["oauth_token"] = oauthToken
	params["oauth_verifier"] = userToken
	params["oauth_version"] = "1.0"

	var apiSignature = createApiSignature(oauth_exchange_token_url, "GET", params, secrets.Secret, &tokenSecret)

	var requestUrl = oauth_exchange_token_url + "?"

	for key, element := range params {
		requestUrl += key + "=" + element + "&"
	}

	requestUrl = strings.TrimRight(requestUrl, "&")
	requestUrl += "&oauth_signature=" + apiSignature

	return requestUrl
}

// Generates the url used to request a token during oauth handshaking
func generateRequestTokenUrl() string {

	secrets := loadOAuthSecrets()

	params := make(map[string]string)
	params["oauth_callback"] = "oob"
	params["oauth_consumer_key"] = secrets.ConsumerKey
	params["oauth_nonce"] = generateNonce()
	params["oauth_signature_method"] = "HMAC-SHA1"
	params["oauth_timestamp"] = fmt.Sprintf("%v", time.Now().Unix())
	params["oauth_version"] = "1.0"

	var apiSignature = createApiSignature(oauth_request_token_url, "GET", params, secrets.Secret, nil)

	var requestUrl = oauth_request_token_url + "?"

	for key, element := range params {
		requestUrl += key + "=" + element + "&"
	}

	requestUrl = strings.TrimRight(requestUrl, "&")
	requestUrl += "&oauth_signature=" + apiSignature

	return requestUrl
}


// Creates the api signature for flickr's oauth implementation
func createApiSignature(
	baseUrl string,
	method string,
	params map[string]string,
	secret string,
	tokenSecret *string) string {

	// Start with the method and url-encoded base url
	var sigBase = method + "&" + url.QueryEscape(baseUrl) + "&"

	// Add the url-encoded params
	for key, element := range params {
		sigBase += url.QueryEscape(key + "=" + element + "&")
	}

	// Remove the last url-encoded '&'
	sigBase = strings.TrimRight(sigBase, "%26")

	// Create the HMAC key
	var hmacKey = secret + "&"
	if tokenSecret != nil {
		hmacKey += *tokenSecret
	}

	// create the hash using the hmac key and the signature base string
	mac := hmac.New(sha1.New, []byte(hmacKey))
	mac.Write([]byte(sigBase))
	expectedMAC := mac.Sum(nil)

	// Encode the result in base64
	en := base64.StdEncoding
	d := make([]byte, en.EncodedLen(len(expectedMAC)))
	en.Encode(d, expectedMAC)

	// Make a string from the bytes
	result := string(d)

	return result
}

// Generates an oauth nonce, just the unix timestamp right now.
func generateNonce() string {

	// Just use the current time
	oauth_nonce = fmt.Sprintf("%v", time.Now().Unix())
	return oauth_nonce
}
