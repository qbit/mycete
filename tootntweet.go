package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

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
		if media_ids, _ := getImagesForTweet(client, matrixnick); media_ids != nil {
			v.Set("media_ids", strings.Join(media_ids, ","))
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

func getImagesForTweet(client *anaconda.TwitterApi, nick string) ([]string, error) {
	imagepaths, err := getUserFileList(nick)
	if err != nil {
		return nil, err
	}
	if len(imagepaths) == 0 {
		return nil, fmt.Errorf("No stored image for nick")
	}
	media_ids := make([]string, len(imagepaths))
	for idx, imagepath := range imagepaths {
		if b64data, err := readFileIntoBase64(imagepath); err != nil {
			return nil, err
		} else {
			if tmedia, err := client.UploadMedia(b64data); err != nil {
				return nil, err
			} else {
				media_ids[idx] = strconv.FormatInt(tmedia.MediaID, 10)
			}
		}

	}
	return media_ids, nil
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
	var mids []mastodon.ID
	usertoot := &mastodon.Toot{Status: post}
	if c.GetValueDefault("images", "enabled", "false") == "true" {
		if mids, err = getImagesForToot(client, matrixnick); err == nil && mids != nil {
			usertoot.MediaIDs = mids
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

func getImagesForToot(client *mastodon.Client, matrixnick string) ([]mastodon.ID, error) {
	imagepaths, err := getUserFileList(matrixnick)
	if err != nil {
		return nil, err
	}
	if len(imagepaths) == 0 {
		return nil, fmt.Errorf("No stored image for nick")
	}
	mastodon_ids := make([]mastodon.ID, len(imagepaths))
	for idx, imagepath := range imagepaths {
		if attachment, err := client.UploadMedia(context.Background(), imagepath); err != nil {
			return nil, err
		} else {
			mastodon_ids[idx] = attachment.ID
		}
	}
	return mastodon_ids, nil
}
