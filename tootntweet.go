package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/ChimeraCoder/anaconda"
	mastodon "github.com/mattn/go-mastodon"
)

/////////////
/// Twitter
/////////////

func initTwitterClient() *anaconda.TwitterApi {
	return anaconda.NewTwitterApiWithCredentials(
		c["twitter"]["access_token"],
		c["twitter"]["access_secret"],
		c["twitter"]["consumer_key"],
		c["twitter"]["consumer_secret"])

}

func sendTweet(client *anaconda.TwitterApi, post, matrixnick string) error {
	var err error
	v := url.Values{}
	v.Set("status", post)
	if c.GetValueDefault("images", "enabled", "false") == "true" {
		if mediaid, _ := getImageForTweet(client, matrixnick); mediaid != 0 {
			v.Set("media_ids", strconv.FormatInt(mediaid, 10))
		}
	}
	// log.Println("sendTweet", post, v)
	_, err = client.PostTweet(post, v)
	return err
}

func getImageForTweet(client *anaconda.TwitterApi, nick string) (int64, error) {
	b64data, err := readFileIntoBase64(hashNickToPath(nick))
	if err != nil {
		return 0, err
	}

	tmedia, err := client.UploadMedia(b64data)
	if err == nil {
		return tmedia.MediaID, err
	} else {
		return 0, err
	}
}

/////////////
/// Mastodon
/////////////

func initMastodonClient() *mastodon.Client {
	return mastodon.NewClient(&mastodon.Config{
		Server:       c["mastodon"]["server"],
		ClientID:     c["mastodon"]["client_id"],
		ClientSecret: c["mastodon"]["client_secret"],
		AccessToken:  c["mastodon"]["access_token"],
	})
}

func sendToot(client *mastodon.Client, post, matrixnick string) (err error) {
	var mid mastodon.ID
	usertoot := &mastodon.Toot{Status: post}
	if c.GetValueDefault("images", "enabled", "false") == "true" {
		if mid, err = getImageForToot(client, matrixnick); err == nil {
			usertoot.MediaIDs = []mastodon.ID{mid}
		}
	}
	// log.Println("sendToot", usertoot)
	_, err = client.PostStatus(context.Background(), usertoot)
	return err
}

func getImageForToot(client *mastodon.Client, matrixnick string) (mastodon.ID, error) {
	imagepath := hashNickToPath(matrixnick)
	if _, err := os.Stat(imagepath); !os.IsNotExist(err) {
		attachment, err := client.UploadMedia(context.Background(), imagepath)
		return attachment.ID, err
	} else if err != nil {
		return mastodon.ID(0), err
	}
	return mastodon.ID(0), fmt.Errorf("No stored image for nick")
}
