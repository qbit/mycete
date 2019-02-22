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

const character_limit_twitter_ int = 280
const character_limit_mastodon_ int = 500
const imgbytes_limit_twitter_ int64 = 5242880
const imgbytes_limit_mastodon_ int64 = 4 * 1024 * 1024

const webbaseformaturl_twitter_ string = "https://twitter.com/statuses/%s"

func checkCharacterLimit(status string) error {
	// get minimum character limit
	climit := 10000
	if c["server"]["mastodon"] == "true" && climit > character_limit_mastodon_ {
		climit = character_limit_mastodon_
	}
	if c["server"]["twitter"] == "true" && climit > character_limit_twitter_ {
		climit = character_limit_twitter_
	}

	// get number of characters ... this is not entirely accurate, but close enough. (read twitters API page on character counting)
	if len(status) <= climit {
		return nil
	} else {
		return fmt.Errorf("status/tweet of %d characters exceeds limit of %d", len(status), climit)
	}
}

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

func sendTweet(client *anaconda.TwitterApi, post, matrixnick string) (weburl string, statusid int64, err error) {
	v := url.Values{}
	v.Set("status", post)
	if c.GetValueDefault("images", "enabled", "false") == "true" {
		if mediaid, _ := getImageForTweet(client, matrixnick); mediaid != 0 {
			v.Set("media_ids", strconv.FormatInt(mediaid, 10))
		}
	}
	// log.Println("sendTweet", post, v)
	var tweet anaconda.Tweet
	tweet, err = client.PostTweet(post, v)
	if err == nil {
		weburl = fmt.Sprintf(webbaseformaturl_twitter_, tweet.IdStr)
		statusid = tweet.Id
	}
	return
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

func sendToot(client *mastodon.Client, post, matrixnick string) (weburl string, statusid mastodon.ID, err error) {
	var mid mastodon.ID
	usertoot := &mastodon.Toot{Status: post}
	if c.GetValueDefault("images", "enabled", "false") == "true" {
		if mid, err = getImageForToot(client, matrixnick); err == nil {
			usertoot.MediaIDs = []mastodon.ID{mid}
		}
	}
	// log.Println("sendToot", usertoot)
	var mstatus *mastodon.Status
	mstatus, err = client.PostStatus(context.Background(), usertoot)
	if mstatus != nil && err == nil {
		weburl = mstatus.URL
		statusid = mstatus.ID
	}
	return
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
