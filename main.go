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

var c goconfig.ConfigMap
var err error

func notify(client *gomatrix.Client, from, msg string) {
	log.Printf("%s: %s\n", from, msg)
	client.SendText(c["matrix"]["room_id"], msg)
}

func sendTweet(client *twitter.Client, post string) error {
	_, _, err = client.Statuses.Update(post, nil)
	return err
}

func sendToot(client *mastodon.Client, post string) error {
	_, err := client.PostStatus(context.Background(), &mastodon.Toot{
		Status: post,
	})
	return err
}

func main() {
	cfile := flag.String("conf", "/etc/mycete.conf", "Configuration file")
	flag.Parse()

	protector.Protect("stdio rpath wpath inet dns")

	c, err = goconfig.ParseFile(*cfile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cli, _ := gomatrix.NewClient(c["matrix"]["url"], "", "")
	resp, err := cli.Login(&gomatrix.ReqLogin{
		Type:     "m.login.password",
		User:     c["matrix"]["user"],
		Password: c["matrix"]["password"],
	})

	mclient := mastodon.NewClient(&mastodon.Config{
		Server:       c["mastodon"]["server"],
		ClientID:     c["mastodon"]["client_id"],
		ClientSecret: c["mastodon"]["client_secret"],
		AccessToken:  c["mastodon"]["access_token"],
	})

	config := oauth1.NewConfig(c["twitter"]["consumer_key"], c["twitter"]["consumer_secret"])
	token := oauth1.NewToken(c["twitter"]["access_token"], c["twitter"]["access_secret"])
	httpClient := config.Client(oauth1.NoContext, token)
	tclient := twitter.NewClient(httpClient)

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
		if ev.Sender == c["matrix"]["user"] {
			return
		}
		if mtype, ok := ev.MessageType(); ok {
			log.Println(ev.Sender)
			switch mtype {
			case "m.text":
				if post, ok := ev.Body(); ok {
					log.Printf("Message: '%s'", post)

					if c["server"]["mastodon"] == "true" {
						err = sendToot(mclient, post)
						if err != nil {
							log.Println(err)
						}
						notify(cli, "mastodon", "sent toot!")
					}

					if c["server"]["twitter"] == "true" {
						err = sendTweet(tclient, post)
						if err != nil {
							log.Println(err)
						}
						notify(cli, "mastodon", "sent toot!")
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
