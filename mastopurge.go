package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	Pagelimit = 40
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

	for {
		res, geterr := c.Client.Do(req)
		if geterr != nil {
			log.Fatal(geterr)
		}

		body, err = ioutil.ReadAll(res.Body)
		if err != nil {
			log.Fatal(err)
		}

		// Only exit if request was not API rate limited
		if rateLimited(res) == false {
			break
		}
	}

	return body, nil
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

/*
 * Helper function to easily read input from console
 */
func readFromConsole() (stringstore string) {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		stringstore = scanner.Text()
	}
	return
}

// Given an array of statuses, returns an array of their ids
func getStatusIds(vs []Status) []uint64 {
	vsm := make([]uint64, len(vs))
	for i, v := range vs {
		vsm[i] = v.ID
	}
	return vsm
}

// Returns true if `a` is an element of the array
func idInSlice(a uint64, list []uint64) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false

func writeSettings(s MastoPurgeSettings) {
	// Write settings to config file
	var config_raw, _ = json.Marshal(s)
	var err = ioutil.WriteFile(".mastopurgesettings", config_raw, 0600)
	if err != nil {
		log.Fatal(err)
	}
}

/*
 * Various data structs
 */

type MastoPurgeSettings struct {
	Server       string `json:"server"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token"`
	MaxAgeHours  int    `json:"max_age_hours,omitempty"`
}

type AccountInfo struct {
	ID       int    `json:"id,string"`
	Username string `json:"username"`
}

type RespAppRegister struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type RespAccessToken struct {
	AccessToken string `json:"access_token"`
}

type Status struct {
	ID uint64 `json:"id,string"`
	//Content     string  `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	var err error

	/*
	 * Add fancyness!
	 */
	fmt.Println("\nWelcome to ...")
	fmt.Println("                     _                                   ")
	fmt.Println(" _ __ ___   __ _ ___| |_ ___  _ __  _   _ _ __ __ _  ___ ")
	fmt.Println("| '_ ` _ \\ / _` / __| __/ _ \\| '_ \\| | | | '__/ _` |/ _ \\")
	fmt.Println("| | | | | | (_| \\__ \\ || (_) | |_) | |_| | | | (_| |  __/")
	fmt.Println("|_| |_| |_|\\__,_|___/\\__\\___/| .__/ \\__,_|_|  \\__, |\\___|")
	fmt.Println("                             |_|              |___/      ")
	fmt.Println("    ... add German Datenhygiene to your Mastodon-Account!")
	fmt.Print("\n\n")

	log.Println("Version 1.1.0")

	/*
	 * Set up settings and Httpclient
	 */
	settings := MastoPurgeSettings{}
	hc := &APIClient{}
	hc.Timeout = time.Second * 5
	hc.Init()

	/*
	 * Check if configuration file .mastopurgesettings exists
	 * - Create if it does not exist
	 * - Use if exists
	 */
	config_raw, readerr := ioutil.ReadFile(".mastopurgesettings")
	if readerr != nil {
		log.Println("MastoPurge configuration file .mastopurgesettings does not exist or is not accessible.")
		fmt.Println("\nFirst we need to connect MastoPurge to your Mastodon account.")
		fmt.Println("Enter the domain of your Mastodon home instance: (e.g. \"metalhead.club\")")
		fmt.Print("[Mastodon home instance]: ")
		settings.Server = readFromConsole()
		hc.Server = "https://" + settings.Server

		// Register application for user on server
		log.Println(">>>>>> Registering MastoPurge App on " + settings.Server)
		params := url.Values{}
		params.Add("client_name", "MastoPurge")
		params.Add("redirect_uris", "urn:ietf:wg:oauth:2.0:oob")
		params.Add("scopes", "read write")
		body, registerErr := hc.Request(http.MethodPost, "/api/v1/apps", params)
		if registerErr != nil {
			log.Fatal(registerErr)
		}

		// Parse response JSON
		respAppRegister := RespAppRegister{}
		err = json.Unmarshal(body, &respAppRegister)
		if err != nil {
			log.Fatal(err)
		}
		settings.ClientID = respAppRegister.ClientID
		settings.ClientSecret = respAppRegister.ClientSecret

		// User must manually authenticate app
		authurl := "https://" + settings.Server + "/oauth/authorize?scope=read%20write&response_type=code&redirect_uri=urn:ietf:wg:oauth:2.0:oob&client_id=" + settings.ClientID
		fmt.Println("\n\nPlease visit this URL in your webbrowser:")
		fmt.Println(authurl)
		fmt.Println("\n\n... and enter the code here:")
		fmt.Print("[Auth code]: ")
		code := readFromConsole()

		// Request auth token via auth code ...
		params = url.Values{}
		params.Add("client_id", settings.ClientID)
		params.Add("client_secret", settings.ClientSecret)
		params.Add("grant_type", "authorization_code")
		params.Add("redirect_uri", "urn:ietf:wg:oauth:2.0:oob")
		params.Add("code", code)
		body, err = hc.Request(http.MethodPost, "/oauth/token", params)
		if err != nil {
			log.Fatal(err)
		}

		respAccessToken := RespAccessToken{}
		err = json.Unmarshal(body, &respAccessToken)
		if err != nil {
			log.Fatal(err)
		}
		settings.AccessToken = respAccessToken.AccessToken
		hc.AccessToken = settings.AccessToken
		writeSettings(settings)
	} else {
		log.Println("MastoPurge configuration found! Reading config.")

		/*
		 * Load settings
		 */
		err = json.Unmarshal(config_raw, &settings)
		if err != nil {
			log.Fatal("Config file is malformed :(\nPlease consider deleting .mastopurgesettings from your file system.")
		}

		hc.Server = "https://" + settings.Server
		hc.AccessToken = settings.AccessToken
	}

	/*
	 * Check if account access is okay
	 */

	log.Println("Requesting access to Mastodon account")
	body, accessErr := hc.Request(http.MethodGet, "/api/v1/accounts/verify_credentials", nil)
	if accessErr != nil {
		log.Fatal(accessErr)
	} else {
		accountinfo := AccountInfo{}
		err = json.Unmarshal(body, &accountinfo)

		if accountinfo.ID == 0 {
			log.Println("Access DENIED :-(")
			log.Fatal("Unfortunately API access was not granted. Consider deleting the .mastopurgesettings and starting MastoPurge again!")
		} else {
			log.Println("Access GRANTED :-)")
			fmt.Println(">>> Account ID:", accountinfo.ID)
			fmt.Println(">>> Username:", accountinfo.Username)

			// Do some date calculations ...
			var maxage time.Duration
			var maxtime time.Time

			if settings.MaxAgeHours != 0 {
				maxage = time.Hour * time.Duration(settings.MaxAgeHours)
			} else {
				for {
					fmt.Println("\nEnter the maximum age of the posts you want to KEEP, e.g. \"30 days\". Older posts will be deleted. Allowed units: hours, days, weeks, months.")
					fmt.Print("[Maximum post age]: ")
					maxagestring := readFromConsole()
					parts := strings.Split(maxagestring, " ")

					if len(parts) == 2 {
						maxagenum, converr := strconv.Atoi(parts[0])
						if converr == nil {
							var factor time.Duration

							switch parts[1] {
							case "hours":
								factor = time.Hour
							case "days":
								factor = time.Hour * time.Duration(24)
							case "weeks":
								factor = time.Hour * time.Duration(24*7)
							case "months":
								factor = time.Hour * time.Duration(24*30)
							default:
								factor = 0
							}

							if factor != 0 {
								maxage = factor * time.Duration(maxagenum)
								settings.MaxAgeHours = int(maxage / time.Hour)
								writeSettings(settings)
								break
							}
						}
					}
					fmt.Println("Error: Invalid age format.")
				}
			}

			maxtime = time.Now().Add(-maxage)
			fmt.Println("Okay, let's do it! ")
			fmt.Println("Posts older than", maxtime, "will be deleted!")
			fmt.Print("Loading gun ")
			for i := 0; i < 40; i++ {
				fmt.Print(".")
				time.Sleep(time.Duration(50) * time.Millisecond)
			}
			time.Sleep(time.Duration(2) * time.Second)
			fmt.Print("\n\n")

			var maxid uint64 = 0
			var deletedcount uint16

			// Get IDs of pinned posts (these won't be deleted)
			log.Printf("========== Fetching pinned statuses ==========\n")
			params := url.Values{}
			params.Add("pinned", "true")
			resp, fetchErr := hc.Request(http.MethodGet, "/api/v1/accounts/"+strconv.Itoa(accountinfo.ID)+"/statuses", params)
			if fetchErr != nil {
				log.Fatal(fetchErr)
			}
			var pinnedStatuses []Status
			err = json.Unmarshal(resp, &pinnedStatuses)
			if err != nil {
				// Maybe server response is an error message?
				fmt.Println(string(resp))
				log.Fatal(err)
			}
			var pinnedStatusIds = getStatusIds(pinnedStatuses)

			// Fetch new pages until there are no more pages
			for {
				log.Printf("========== Fetching new statuses until status %d ==========\n", maxid)

				nodeletions := true

				// Fetch posts
				params := url.Values{}
				params.Add("limit", strconv.Itoa(Pagelimit))
				if maxid != 0 {
					params.Add("max_id", fmt.Sprint(maxid))
				}
				resp, fetchErr := hc.Request(http.MethodGet, "/api/v1/accounts/"+strconv.Itoa(accountinfo.ID)+"/statuses", params)
				if fetchErr != nil {
					log.Fatal(fetchErr)
				}

				var statuses []Status
				err = json.Unmarshal(resp, &statuses)
				if err != nil {
					// Maybe server response is an error message?
					fmt.Println(string(resp))
					log.Fatal(err)
				}

				// Exit killer loop if there are no more statuses
				if len(statuses) == 0 {
					break
				}

				for _, status := range statuses {
					// Parse time
					if status.CreatedAt.Before(maxtime) {
						if idInSlice(status.ID, pinnedStatusIds) {
							log.Println("Status " + fmt.Sprint(status.ID) + " is pinned; not deleting.")
							continue
						}
						// Delete post
						nodeletions = false
						delResp, delErr := hc.Request(http.MethodDelete, "/api/v1/statuses/"+fmt.Sprint(status.ID), nil)
						if delErr != nil {
							log.Println("!!! Could not delete status " + fmt.Sprint(status.ID) + " !!!")
						}

						if string(delResp) == "{}" {
							//log.Println("Status " + fmt.Sprint(status.ID) + " successfully deleted!")
							deletedcount++
						} else {
							log.Println("Status " + fmt.Sprint(status.ID) + " could not be deleted :(")
						}
					}

					if status.ID < maxid || maxid == 0 {
						maxid = status.ID - 1
					}
				}

				if nodeletions {
					log.Println("No posts to be deleted on this page. Trying next page ...")
				} else {
					log.Println(deletedcount, "statuses deleted.")
					// Wait before fetching a new page. Give server time to re-assemble pages.
					time.Sleep(time.Duration(1) * time.Second)
				}
			}

			// No more pages, done! :-)
			fmt.Println(">>>>>> ", deletedcount, "statuses were successfully deleted.")
		}
	}
}
