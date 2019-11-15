package main

import (
	"bufio"
	"encoding/json"
	"flag"
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
}

/*
 * Various data structs
 */

type MastoPurgeSettings struct {
	Server       string `json:"server"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token"`
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

var (
	noninteractiveMode = flag.Bool("noninteractive", false, "Run in non-interactive mode, suitable for eg. cron jobs. (On first run with a missing settings file, the config process will run interactively.)")
	maxAgeArgument     = flag.String("maxAge", "", "Max age of posts you want to keep. Required when running in non-interactive mode. Allowed units: hours, days, weeks, months, years. Example: \"6 months\".")
	configFile         = flag.String("config", ".mastopurgesettings", "Path + filename for the settings file.")
	printVersion       = flag.Bool("version", false, "Print version, and exit.")
	dryRun             = flag.Bool("dryRun", false, "Run MastoPurge to preview its results, but without actually deleting any statuses.")
)

const MastoPurgeVersion = "1.1.0"

func main() {
	flag.Parse()

	if *printVersion {
		fmt.Printf("MastoPurge version %s\n", MastoPurgeVersion)
		os.Exit(0)
	}

	interactiveMode := !(*noninteractiveMode)

	if interactiveMode {
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
		fmt.Printf("Version %s\n\n", MastoPurgeVersion)
	}

	/*
	 * Set up settings and Httpclient
	 */
	settings := MastoPurgeSettings{}
	hc := &APIClient{}
	hc.Timeout = time.Second * 5
	hc.Init()

	/*
	 * Check if configuration file (*configFile, default .mastopurgesettings) exists
	 * - Create if it does not exist
	 * - Use if exists
	 */
	configRaw, readerr := ioutil.ReadFile(*configFile)
	if readerr != nil {
		log.Printf("MastoPurge configuration file %s does not exist or is not accessible.\n", *configFile)
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
		err := json.Unmarshal(body, &respAppRegister)
		if err != nil {
			log.Fatal(err)
		}
		settings.ClientID = respAppRegister.ClientID
		settings.ClientSecret = respAppRegister.ClientSecret

		// User must manually authenticate app
		authurl := "https://" + settings.Server + "/oauth/authorize?scope=read%20write&response_type=code&redirect_uri=urn:ietf:wg:oauth:2.0:oob&client_id=" + settings.ClientID
		fmt.Println("\nPlease visit this URL in your web browser:")
		fmt.Println(authurl)
		fmt.Println("\n... and enter the code here:")
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

		// Write settings to config file
		configRaw, _ = json.Marshal(settings)
		err = ioutil.WriteFile(*configFile, configRaw, 0600)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		log.Println("MastoPurge configuration found! Reading config.")

		/*
		 * Load settings
		 */
		err := json.Unmarshal(configRaw, &settings)
		if err != nil {
			log.Fatalf("Config file is malformed :(\nPlease consider deleting '%s' from your file system and starting MastoPurge again.", *configFile)
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
		err := json.Unmarshal(body, &accountinfo)

		if accountinfo.ID == 0 {
			log.Println("Access DENIED :-(")
			log.Fatalf("Unfortunately API access was not granted. Consider deleting '%s' and starting MastoPurge again!", *configFile)
		} else {
			log.Println("Access GRANTED :-)")
			log.Println(">>> Account ID:", accountinfo.ID)
			log.Println(">>> Username:", accountinfo.Username)

			// Do some date calculations ...
			var maxage time.Duration
			var maxtime time.Time

			for {
				maxagestring := *maxAgeArgument
				if maxagestring == "" {
					if interactiveMode {
						fmt.Println("\nEnter the maximum age of the posts you want to KEEP, e.g. \"30 days\". Older posts will be deleted. Allowed units: hours, days, weeks, months, years.")
						fmt.Print("[Maximum post age]: ")
						maxagestring = readFromConsole()
					} else {
						log.Println("missing required argument -maxAge")
						flag.PrintDefaults()
						os.Exit(1)
					}
				}

				parts := strings.Split(maxagestring, " ")

				if len(parts) == 2 {
					maxagenum, converr := strconv.Atoi(parts[0])
					if converr == nil {
						var factor time.Duration

						switch parts[1] {
						case "hours", "hour":
							factor = time.Hour
						case "days", "day":
							factor = time.Hour * time.Duration(24)
						case "weeks", "week":
							factor = time.Hour * time.Duration(24*7)
						case "months", "month":
							factor = time.Hour * time.Duration(24*30)
						case "years", "year":
							factor = time.Hour * time.Duration(24*365)
						default:
							factor = 0
						}

						if factor != 0 {
							maxage = factor * time.Duration(maxagenum)
							break
						}
					}
				}

				if interactiveMode {
					fmt.Println("Error: Invalid age format.")
				} else {
					log.Fatalf("Invalid maximum age \"%s\".", maxagestring)
				}
			}

			maxtime = time.Now().Add(-maxage)
			if interactiveMode {
				fmt.Println("Okay, let's do it! ")
				fmt.Println("Posts older than", maxtime, "will be deleted!")
				fmt.Print("Loading gun ")
				for i := 0; i < 40; i++ {
					fmt.Print(".")
					time.Sleep(time.Duration(50) * time.Millisecond)
				}
				time.Sleep(time.Duration(2) * time.Second)
				fmt.Print("\n\n")
			} else {
				log.Println("Posts older than", maxtime.Format("Jan 2, 2006 at 3:04:05 PM MST"), "will be deleted!")
			}

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
				log.Println(string(resp))
				log.Fatal(err)
			}
			pinnedStatusIds := getStatusIds(pinnedStatuses)
			log.Printf("Found %d pinned statuses, which will not be deleted.", len(pinnedStatusIds))

			var maxid uint64 = 0
			var deletedcount uint16

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
					log.Println(string(resp))
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

						if !*dryRun {
							delResp, delErr := hc.Request(http.MethodDelete, "/api/v1/statuses/"+fmt.Sprint(status.ID), nil)
							if delErr != nil {
								log.Println("!!! Could not delete status " + fmt.Sprint(status.ID) + " !!!")
							}

							var delStatus Status
							err = json.Unmarshal(delResp, &delStatus)
							if err != nil {
								log.Println(string(delResp))
								log.Fatal(err)
							}

							if delStatus.ID == status.ID {
								deletedcount++
							} else {
								log.Println("Status " + fmt.Sprint(status.ID) + " could not be deleted :( \nResponse: " + string(delResp))
							}
						}
					}

					if status.ID < maxid || maxid == 0 {
						maxid = status.ID - 1
					}
				}

				if nodeletions {
					log.Println("No posts to be deleted on this page. Trying next page ...")
				} else {
					if deletedcount == 0 && *dryRun {
						log.Println("0 statuses deleted, because -dryRun was passed.")
					} else {
						log.Println(deletedcount, "statuses deleted.")
					}
					// Wait before fetching a new page. Give server time to re-assemble pages.
					time.Sleep(time.Duration(1) * time.Second)
				}
			}

			if deletedcount == 0 && *dryRun {
				log.Println("[dryRun] 0 statuses deleted in total, because -dryRun was passed.")
			}

			// No more pages, done! :-)
			if interactiveMode {
				fmt.Println(">>>>>> ", deletedcount, "statuses were successfully deleted.")
			} else {
				log.Println(deletedcount, "statuses were successfully deleted.")
			}
		}
	}
}
