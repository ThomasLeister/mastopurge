package main

import (
    "fmt"
    "github.com/mattn/go-mastodon"
    "context"
    "log"
    "time"
    "bufio"
    "os"
    "strings"
)


func main() {
    var err error
    fmt.Println("Hello!")
    fmt.Println("MastoPurge deletes your outdated Mastodon posts via the API.")
    fmt.Println("==================================================================")
    fmt.Println("Please note that this can take some time due to API rate limits!")
    fmt.Println("==================================================================")

    reader := bufio.NewReader(os.Stdin)
    var instance string
    var email string
    var password string
    var client_id string
    var client_secret string

    fmt.Println("Your instance: (e.g. \"social.tchncs.de\")")
    instance, _ = reader.ReadString('\n')
    instance = strings.Replace(instance, "\n", "", -1)


    // Registering application
    app, err := mastodon.RegisterApp(context.Background(), &mastodon.AppConfig{
		Server:     "https://" + instance,
		ClientName: "Mastopurge",
		Scopes:     "read write follow",
		Website:    "https://github.com/ThomasLeister/mastopurge",
	})
	if err != nil {
		log.Fatal(err)
	}

	//fmt.Printf("client-id    : %s\n", app.ClientID)
	//fmt.Printf("client-secret: %s\n", app.ClientSecret)
    client_id = app.ClientID
    client_secret = app.ClientSecret


    fmt.Println("Your account's e-mail address:")
    email, _ = reader.ReadString('\n')
    email = strings.Replace(email, "\n", "", -1)

    fmt.Println("Your password:")
    password, _ = reader.ReadString('\n')
    password = strings.Replace(password, "\n", "", -1)

    fmt.Println("Maximum age of posts (in days - older posts are deleted!):")
    var maxage int
    fmt.Scan(&maxage)

    c := mastodon.NewClient(&mastodon.Config{
		Server:       "https://" + instance,
		ClientID:     client_id,
		ClientSecret: client_secret,
	})

    log.Println("Authorizing ...")
    log.Println("Email:", email)
    log.Println("Password:", password)

    err = c.Authenticate(context.Background(), email, password)
	if err != nil {
		log.Fatal("Failed to authenticate", err)
	}


    // Calc time things
    // Maximum age
    var maxtime = time.Now().Add(-1 * (time.Duration(maxage) * time.Hour * 24))
    log.Println("Now time:", time.Now())
    log.Println("Maximum time:", maxtime)


    // Get user account
    account, accerr := c.GetAccountCurrentUser(context.Background())
    if accerr != nil {
        log.Fatal(accerr)
    }

    var maxid int64
    maxid = 1000000000

    for {
        log.Println("Getting account statuses until ID", maxid)
        var pg mastodon.Pagination
        pg.Limit = 40
        pg.MaxID = maxid

    	timeline, err := c.GetAccountStatuses(context.Background(), account.ID, &pg)
    	if err != nil {
    		log.Fatal(err)
    	}

        if len(timeline) == 0 {
            break
        }

    	for i := len(timeline) - 1 ; i >= 0; i-- {
            status := timeline[i]
            log.Println("Checking status", status.ID)

            // Check if this status is outdated
            if status.CreatedAt.Before(maxtime) {
                // Delete this status
                log.Println("Deleting outdated status...")
                err := c.DeleteStatus(context.Background(), status.ID)
                if err != nil {
                    log.Fatal(err)
                }
                log.Println("Status", status.ID, "deleted.")
                time.Sleep(1 * time.Second)
            }

            if status.ID < maxid {
                maxid = status.ID
            }
    	}
    }

    log.Println("Checked all statuses. Finished.")

}
