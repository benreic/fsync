package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

var oauth_request_token_url = "https://www.flickr.com/services/oauth/request_token"
var oauth_exchange_token_url = "https://www.flickr.com/services/oauth/access_token"
var oauth_nonce = ""
var cacheFile = "oauth.json"
var oauthSecretsFile = "oauth-secrets.json"

type FlickrOAuth struct {
	FullName         string
	OAuthToken       string
	OAuthTokenSecret string
	UserNSID         string
	Username         string
}

type OAuthSecrets struct {
	ConsumerKey  string
	Secret       string
	MinitokenUrl string
}

func (s OAuthSecrets) isValid() bool {

	if len(s.ConsumerKey) == 0 ||
		len(s.Secret) == 0 ||
		len(s.MinitokenUrl) == 0 {
		return false
	}

	return true
}

/**
 * Loads the secret OAuth data for this app from a json file. This file
 * should not get committed to source control.
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @return  OAuthSecrets
**/

func loadOAuthSecrets() OAuthSecrets {

	s := new(OAuthSecrets)
	filePath := getUserFilePath(oauthSecretsFile)
	if pathExists(filePath) {
		fileContents, _ := ioutil.ReadFile(filePath)
		if len(fileContents) > 0 {
			json.Unmarshal(fileContents, &s)
		} else {
			logMessage("oauth-secrets.json file was empty", false)
		}
	} else {
		msg := fmt.Sprintf("No oauth secrets file found at: %v", filePath)
		logMessage(msg, false)
	}

	return *s
}

/**
 * Checks for cached OAuth credentials so we don't need
 * to go through the OAuth process again.
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @return  FlickrOAuth
**/

func checkForExistingOAuthCredentials() FlickrOAuth {

	var oauth = new(FlickrOAuth)
	filePath := getUserFilePath(cacheFile)
	if pathExists(filePath) {
		fileContents, _ := ioutil.ReadFile(filePath)
		if len(fileContents) > 0 {
			json.Unmarshal(fileContents, &oauth)
		}
	}

	return *oauth
}

/**
 * Does the OAuth handshaking between Flickr and the user, if
 * we didn't find any cached credentials.
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @return  FlickrOAuth
**/

func doOAuthSetup() FlickrOAuth {

	oauthResult := FlickrOAuth{"", "", "", "", ""}

	// Get the response from the request token url
	// and check for errors
	body, err := makeGetRequest(func() string { return generateRequestTokenUrl() })

	if err != nil {
		logMessage(fmt.Sprintf("Hmm, something went wrong: %v", err), true)
		return oauthResult
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
		logMessage("An error occurred, there was no token: "+string(body), false)
		return oauthResult
	}

	if oauth_token_secret == "" {
		logMessage("An error occurred, there was no secret: "+string(body), false)
		return oauthResult
	}

	// Send the user to flickr to authorize us
	url := "https://www.flickr.com/services/oauth/authorize?perms=read&oauth_token=" + oauth_token
	switch runtime.GOOS {
	case "linux":
		exec.Command("xdg-open", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	case "windows":
		exec.Command(`C:\Windows\System32\rundll32.exe`, "url.dll,FileProtocolHandler", url).Start()
	}

	// Have them enter the 9 digit code from flickr
	fmt.Println("Authorize the app on flickr's site and enter the nine digit code here and press 'Return':")
	var userToken = ""
	_, err = fmt.Scanln(&userToken)

	// Get the response and check for errors
	body, err = makeGetRequest(func() string { return generateExchangeUrl(userToken, oauth_token, oauth_token_secret) })

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

	filePath := getUserFilePath(cacheFile)
	err = ioutil.WriteFile(filePath, b, perms)
	if err != nil {
		panic(err)
	}
	return oauthResult
}

/**
 * Generates an OAuth url for flickr based on the method the user wants and any extra params
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   string               The base url
 * @param   string               The method (api name) we're using
 * @param   FlickrOAuth          The OAuth app configuration
 * @param   map[string]string    Any extra params for the api call
 * @return  string               The resulting OAuth url for use in a GET request
**/

func generateOAuthUrl(baseUrl string, method string, auth FlickrOAuth, extraParams map[string]string) string {

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

		for key, element := range extraParams {
			params[key] = element
		}
	}

	apiSignature := createApiSignature(baseUrl, "GET", params, secrets.Secret, &auth.OAuthTokenSecret)
	params["oauth_signature"] = apiSignature

	sortedKeys := []string{}

	for key, _ := range params {
		sortedKeys = append(sortedKeys, key)
	}

	sort.Strings(sortedKeys)

	requestUrl := baseUrl + "?"
	for _, key := range sortedKeys {
		requestUrl += key + "=" + params[key] + "&"
	}

	requestUrl = strings.TrimRight(requestUrl, "&")
	return requestUrl
}

/**
 * Generates the exchange token url, used during oauth handshaking
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   string    The user token
 * @param   string    The OAuth token
 * @param   string    The app's token secret
 * @return  string    The url to use to exchange the token
**/

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

/**
 * Generates the url used to request a token during OAuth handshaking
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @return  string    The url to request a token
**/

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

	sortedKeys := []string{}

	for key, _ := range params {
		sortedKeys = append(sortedKeys, key)
	}

	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		requestUrl += key + "=" + params[key] + "&"
	}

	requestUrl = strings.TrimRight(requestUrl, "&")
	requestUrl += "&oauth_signature=" + apiSignature

	return requestUrl
}

/**
 * Creates the api signature for flickr's oauth implementation
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   string                The base url
 * @param   string                The method (api name)
 * @param   map[string]string     Params for the api call
 * @param   string                Secret
 * @param   string                Token secret
 * @return  string                The api call's HMAC secret
**/

func createApiSignature(
	baseUrl string,
	method string,
	params map[string]string,
	secret string,
	tokenSecret *string) string {

	// Start with the method and url-encoded base url
	var sigBase = method + "&" + url.QueryEscape(baseUrl) + "&"

	sortedKeys := []string{}

	for key, _ := range params {
		sortedKeys = append(sortedKeys, key)
	}

	sort.Strings(sortedKeys)

	// Add the sorted, url-encoded params
	for _, key := range sortedKeys {
		sigBase += url.QueryEscape(key + "=" + url.QueryEscape(params[key]) + "&")
	}

	// Remove the last url-encoded '&'
	sigBase = strings.TrimRight(sigBase, "%26")

	return generateSignatureFromString(sigBase, secret, tokenSecret)
}

func generateSignatureFromString(source string, secret string, tokenSecret *string) string {

	// Create the HMAC key
	var hmacKey = secret + "&"
	if tokenSecret != nil {
		hmacKey += *tokenSecret
	}

	// create the hash using the hmac key and the signature base string
	mac := hmac.New(sha1.New, []byte(hmacKey))
	mac.Write([]byte(source))
	expectedMAC := mac.Sum(nil)

	// Encode the result in base64
	en := base64.StdEncoding
	d := make([]byte, en.EncodedLen(len(expectedMAC)))
	en.Encode(d, expectedMAC)

	// Make a string from the bytes
	result := string(d)
	result = url.QueryEscape(result)

	return result
}

/**
 * Create an OAuth nonce to be used for a request, just the Unix timestamp.
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @return  string    The Nonce
**/

func generateNonce() string {

	// Just use the current time
	oauth_nonce = fmt.Sprintf("%v", time.Now().Unix())
	return oauth_nonce
}

/**
 * Generates the api signature for a debug_sbs value that
 * Flickr returns when we have an invalid signature error.
 *
 * @author Ben Reichelt <ben.reichelt@gmail.com>
 *
 * @param   string   debugSbs
 * @return  string
**/

func getApiSignature(debugSbs string) string {

	secrets := loadOAuthSecrets()
	appFlickrOAuth := checkForExistingOAuthCredentials()

	if appFlickrOAuth.OAuthToken == "" {
		logMessage("Can't print api signature, no OAuth credentials exist.", true)
		return ""
	}

	signature := generateSignatureFromString(debugSbs, secrets.Secret, &appFlickrOAuth.OAuthTokenSecret)
	return signature
}
