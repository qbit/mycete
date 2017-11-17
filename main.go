package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/gokyle/goconfig"
	"github.com/matrix-org/gomatrix"
	"github.com/mattn/go-mastodon"
	"github.com/qbit/mycete/protector"
)

func main() {
	cfile := flag.String("config", "/etc/mycete.conf", "Configuration file")
	flag.Parse()

	protector.Protect("stdio rpath wpath inet dns")

	c, err := goconfig.ParseFile(*cfile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	config := oauth1.NewConfig(c["twitter"]["consumer_key"], c["twitter"]["consumer_secret"])
	token := oauth1.NewToken(c["twitter"]["access_token"], c["twitter"]["access_secret"])

	httpClient := config.Client(oauth1.NoContext, token)

	mclient := mastodon.NewClient(&mastodon.Config{
		Server:       c["mastodon"]["server"],
		ClientID:     c["mastodon"]["client_id"],
		ClientSecret: c["mastodon"]["client_secret"],
		AccessToken:  c["mastodon"]["access_token"],
	})

	// Twitter client
	client := twitter.NewClient(httpClient)

	cli, _ := gomatrix.NewClient(c["matrix"]["url"], "", "")
	resp, err := cli.Login(&gomatrix.ReqLogin{
		Type:     "m.login.password",
		User:     c["matrix"]["user"],
		Password: c["matrix"]["password"],
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cli.SetCredentials(resp.UserID, resp.AccessToken)

	if _, err := cli.JoinRoom(c["matrix"]["room_id"], "", nil); err != nil {
		panic(err)
	}

	syncer := cli.Syncer.(*gomatrix.DefaultSyncer)
	syncer.OnEventType("m.room.message", func(ev *gomatrix.Event) {
		if mtype, ok := ev.MessageType(); ok {
			switch mtype {
			case "m.text":
				if post, ok := ev.Body(); ok {
					log.Printf("Message: '%s'", post)

					if c["server"]["mastodon"] == "true" {
						_, err := mclient.PostStatus(context.Background(), &mastodon.Toot{
							Status: post,
						})

						if err != nil {
							log.Println(err)
						}
					}

					if c["server"]["mastodon"] == "true" {
						_, _, err = client.Statuses.Update(post, nil)
						if err != nil {
							log.Println(err)
						}
					}
				}
			default:
				fmt.Printf("%s messages are currently not supported", mtype)
			}

		}

	})

	go func() {
		for {
			log.Println("syncing..")
			if err := cli.Sync(); err != nil {
				fmt.Println("Sync() returned ", err)
			}
			time.Sleep(100 * time.Second)
		}
	}()

	for {
		time.Sleep(100 * time.Second)
	}
}
