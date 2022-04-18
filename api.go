package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// APIClient provides an easy way to interface with the API.
type APIClient struct {
	Server      string
	Timeout     time.Duration
	Client      http.Client
	UserAgent   string
	AccessToken string
}

// Init sets the default UserAgent for the APIClient, and creates the HTTP
// client as well.
func (c *APIClient) Init() {
	c.Client = http.Client{
		Timeout: c.Timeout,
	}
	c.UserAgent = "MastoPurge"
}

// Request makes a new request to the API. method is the HTTP method to use,
// e.g. GET or POST, whereas endpoint is the API endpoint to which we should
// make the request.
func (c *APIClient) Request(method, endpoint string, params url.Values) (body []byte, err error) {
	body, _, err = c.RequestWithLink(method, endpoint, params)
	return body, err
}

// RequestWithLink makes a new request to the API. method is the HTTP method to use,
// e.g. GET or POST, whereas endpoint is the API endpoint to which we should
// make the request.
// Returns the 'Link' header with either an empty string or links to the previous or next chunk.
func (c *APIClient) RequestWithLink(method, endpoint string, params url.Values) (body []byte, linkHeader string, err error) {
	// Set up request: if it's a POST/PUT, we make the body urlencoded.
	uri := c.Server + endpoint
	var req *http.Request
	if method == http.MethodPost || method == http.MethodPut {
		req, err = http.NewRequest(method, uri, strings.NewReader(params.Encode()))
	} else {
		var paramsEncoded string
		if params != nil {
			paramsEncoded = "?" + params.Encode()
		}
		req, err = http.NewRequest(method, uri+paramsEncoded, nil)
	}
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", c.UserAgent)
	if c.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	}

	// Keep going until we get the response,
	// possibly including some waiting around for API rate limits.
	for {
		res, geterr := c.Client.Do(req)
		if geterr != nil {
			log.Fatal(geterr)
		}

		body, err = ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}

		// Fetch 'Link' header. Empty string, if none is available.
		linkHeader = res.Header.Get("Link")
		fmt.Printf("found Link header: %s", linkHeader)

		// Only exit if request was not API rate limited
		if !rateLimited(res) {
			break
		}
	}

	return body, linkHeader, nil
}

// rateLimited checks if API throttling is active. If yes, it waits the time
// defined by the server (or a default of 30 seconds) and repeats the http
// request. Returns whether the request was rate limited.
func rateLimited(res *http.Response) bool {
	// request was not rate limited - nothing to do.
	if res.StatusCode != 429 {
		return false
	}

	var waitDuration time.Duration
	waitUntil, err := time.Parse(time.RFC3339, res.Header.Get("X-Ratelimit-Reset"))
	if err != nil {
		fmt.Println("Cool down time was not defined by server. Waiting for 30 seconds.")
		waitDuration = 30 * time.Second
	} else {
		waitDuration = time.Until(waitUntil)
	}

	fmt.Printf(">>>>>> Server has run hot and is throttling. We have to wait for %d seconds until it has cooled down. Please be patient ...\n", int(waitDuration.Seconds()))
	time.Sleep(waitDuration)

	fmt.Println("Retrying ...")
	return true
}