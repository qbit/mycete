package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/matrix-org/gomatrix"
	mastodon "github.com/mattn/go-mastodon"
)

var (
	mastodon_status_uri_re_ *regexp.Regexp
	twitter_status_uri_re_  *regexp.Regexp
)

func init() {
	mastodon_status_uri_re_ = regexp.MustCompile(`^https?://[^/]+/@\w+/(\d+)$`)
	twitter_status_uri_re_ = regexp.MustCompile(`^https?://twitter\.com/.+/status(?:es)?/(\d+)$`)
}

func mxNotify(client *gomatrix.Client, from, msg string) {
	log.Printf("%s: %s\n", from, msg)
	client.SendText(c["matrix"]["room_id"], msg)
}

// Ignore messages from ourselves
// Ignore messages from rooms we are not interessted in
func mxIgnoreEvent(ev *gomatrix.Event) bool {
	return ev.Sender == c["matrix"]["user"] || ev.RoomID != c["matrix"]["room_id"]
}

func runMatrixPublishBot() {
	mxcli, _ := gomatrix.NewClient(c["matrix"]["url"], "", "")
	resp, err := mxcli.Login(&gomatrix.ReqLogin{
		Type:     "m.login.password",
		User:     c["matrix"]["user"],
		Password: c["matrix"]["password"],
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	mclient := initMastodonClient()
	tclient := initTwitterClient()

	mxcli.SetCredentials(resp.UserID, resp.AccessToken)

	rums_store_chan, rums_retrieve_chan := runRememberUsersMessageToStatus()

	if _, err := mxcli.JoinRoom(c["matrix"]["room_id"], "", nil); err != nil {
		panic(err)
	}

	var markseen_c chan<- mastodon.ID = nil
	if c.SectionInConfig("feed2matrix") {
		markseen_c = taskWriteMastodonBackIntoMatrixRooms(mclient, mxcli)
	}

	syncer := mxcli.Syncer.(*gomatrix.DefaultSyncer)
	syncer.OnEventType("m.room.message", func(ev *gomatrix.Event) {
		if mxIgnoreEvent(ev) { //ignore messages from ourselves or from other rooms in case of dual-login
			return
		}

		if mtype, ok := ev.MessageType(); ok {
			log.Println(ev.Sender)
			switch mtype {
			case "m.text":
				if post, ok := ev.Body(); ok {
					log.Printf("Message: '%s'", post)
					if strings.HasPrefix(post, reblog_cmd_) {
					} else if strings.HasPrefix(post, favourite_cmd_) {
						//TODO: accept strings in form:
						// ✓ url (where we can detect twitter or mastodon)
						// ✓ "toot <ID>" --> mastodon
						// ✓ "status <ID>" --> mastdon
						// ✓ "tweet <ID>" --> twitter
						// ✓ "birdsite <ID>" --> twitter
						// - last --> favorite the last received toot or tweet
						args := strings.SplitN(strings.TrimSpace(post[len(favourite_cmd_):]), " ", 3)
						if len(args) > 1 {
							switch strings.ToLower(args[0]) {
							case "toot", "status":
								if _, err = mclient.Favourite(context.Background(), mastodon.ID(args[1])); err != nil {
									mxNotify(mxcli, "favorite", fmt.Sprintf("error while favouring toot %s: %s", args[1], err.Error()))
									return
								}
							case "tweet", "birdsite":
								postid, err := strconv.ParseInt(args[1], 10, 64)
								if err != nil || postid <= 0 {
									mxNotify(mxcli, "favorite", fmt.Sprintf("Sorry, can't parse that ID given as 2nd argument: %s", err))
									return
								}
								if _, err = tclient.Favorite(postid); err != nil {
									mxNotify(mxcli, "favorite", fmt.Sprintf("error while favouring tweet %d: %s", postid, err.Error()))
									return
								}
							}
						} else if len(args) == 1 {
							if args[0] == "last" {
								mxNotify(mxcli, "favorite", "Sorry, not implemented yet") //TODO
								return
							} else if matchlist := mastodon_status_uri_re_.FindStringSubmatch(args[0]); len(matchlist) >= 2 {
								if _, err = mclient.Favourite(context.Background(), mastodon.ID(matchlist[1])); err != nil {
									mxNotify(mxcli, "favorite", fmt.Sprintf("error while favouring toot %s: %s", matchlist[1], err.Error()))
									return
								}
							} else if matchlist := twitter_status_uri_re_.FindStringSubmatch(args[0]); len(matchlist) >= 2 {
								postid, err := strconv.ParseInt(matchlist[1], 10, 64)
								if err != nil || postid <= 0 {
									mxNotify(mxcli, "favorite", fmt.Sprintf("Sorry, can't parse that ID in the URL: %s", err))
									return
								}
								if _, err = tclient.Favorite(postid); err != nil {
									mxNotify(mxcli, "favorite", fmt.Sprintf("error while favouring tweet %d: %s", postid, err.Error()))
									return
								}
							} else {
								mxNotify(mxcli, "favorite", "Please say "+favourite_cmd_+" followed by 'last', <status URL> or 'toot'/'tweet' <ID>")
								return
							}
						} else {
							mxNotify(mxcli, "favorite", "Please say "+favourite_cmd_+" followed by 'last', <status URL> or 'toot'/'tweet' <ID>")
							return
						}
						mxNotify(mxcli, "favorite", "Ok, I favorited that status for you")
					} else if strings.HasPrefix(post, guard_prefix_) {
						post = strings.TrimSpace(post[len(guard_prefix_):])

						if err = checkCharacterLimit(post); err != nil {
							log.Println(err)
							mxNotify(mxcli, "limitcheck", fmt.Sprintf("Not tweeting/tooting this! %s", err.Error()))
							return
						}

						go func() {
							lock := getPerUserLock(ev.Sender)
							lock.Lock()
							defer lock.Unlock()
							var reviewurl string
							var twitterid int64
							var mastodonid mastodon.ID

							if c["server"]["mastodon"] == "true" {
								reviewurl, mastodonid, err = sendToot(mclient, post, ev.Sender)
								if markseen_c != nil {
									markseen_c <- mastodonid
								}
								if err != nil {
									log.Println("MastodonTootERROR:", err)
									mxNotify(mxcli, "mastodon", "ERROR while tooting!")
								} else {
									mxNotify(mxcli, "mastodon", fmt.Sprintf("sent toot! %s", reviewurl))
								}
							}

							if c["server"]["twitter"] == "true" {
								reviewurl, twitterid, err = sendTweet(tclient, post, ev.Sender)
								if err != nil {
									log.Println("TwitterTweetERROR:", err)
									mxNotify(mxcli, "twitter", "ERROR while tweeting!")
								} else {
									mxNotify(mxcli, "twitter", fmt.Sprintf("sent tweet! %s", reviewurl))
								}
							}

							//remember posted status IDs
							rums_store_chan <- RUMSStoreMsg{key: ev.ID, data: MsgStatusTripple{ev.Sender, mastodonid, twitterid}}

							//remove saved image file if present. We only attach an image once.
							if c.GetValueDefault("images", "enabled", "false") == "true" {
								rmFile(ev.Sender)
							}

						}()
					}
				}
			case "m.image":
				if c.GetValueDefault("images", "enabled", "false") != "true" {
					mxNotify(mxcli, "error", "image support is disabled. Set [images]enabled=true")
					fmt.Println("ignoring image since support not enabled in config file")
					return
				}
				if infomapi, inmap := ev.Content["info"]; inmap {
					if infomap, ok := infomapi.(map[string]interface{}); ok {
						if imgsizei, insubmap := infomap["size"]; insubmap {
							if imgsize, ok2 := imgsizei.(int64); ok2 {
								if err = checkImageBytesizeLimit(imgsize); err != nil {
									mxNotify(mxcli, "imagesaver", err.Error())
									return
								}
							}
						}
					}
				}
				if urli, inmap := ev.Content["url"]; inmap {
					if url, ok := urli.(string); ok {
						go func() {
							lock := getPerUserLock(ev.Sender)
							lock.Lock()
							defer lock.Unlock()
							if err := saveMatrixFile(mxcli, ev.Sender, url); err != nil {
								mxNotify(mxcli, "error", "Could not get your image! "+err.Error())
								fmt.Println("ERROR downloading image:", err)
								return
							}
							mxNotify(mxcli, "imagesaver", fmt.Sprintf("image saved. Will tweet/toot with %s's next message", ev.Sender))
						}()
					}
				}
			case "m.video", "m.audio":
				fmt.Printf("%s messages are currently not supported", mtype)
				mxNotify(mxcli, "runMatrixPublishBot", "Ahh. Audio/Video files are not supported directly. Please just include it's URL in your Toot/Tweet and Mastodon/Twitter will do the rest.")
			default:
				fmt.Printf("%s messages are currently not supported", mtype)
				//remove saved image file if present. We only attach an image once.
			}
		}
	})

	/// Support redactions to "take back an uploaded image"
	syncer.OnEventType("m.room.redaction", func(ev *gomatrix.Event) {
		if mxIgnoreEvent(ev) { //ignore messages from ourselves or from other rooms in case of dual-login
			return
		}
		if c.GetValueDefault("images", "enabled", "false") == "true" {
			go func() {
				lock := getPerUserLock(ev.Sender)
				lock.Lock()
				defer lock.Unlock()
				err := rmFile(ev.Sender)
				if err == nil {
					mxNotify(mxcli, "redaction", fmt.Sprintf("%s's image has been redacted. Next toot/weet will not contain any image.", ev.Sender))
				}
				if err != nil && !os.IsNotExist(err) {
					log.Println("ERROR deleting image:", err)
				}

			}()
		}
		go func() {
			future_chan := make(chan *MsgStatusTripple, 1)
			rums_retrieve_chan <- RUMSRetrieveMsg{key: ev.Redacts, future: future_chan}
			rums_ptr := <-future_chan
			if rums_ptr == nil {
				return
			}
			if c.GetValueDefault("matrix", "admins_can_redact_user_status", "false") == "true" || rums_ptr.MatrixUser == ev.Sender {
				if _, err := tclient.DeleteTweet(rums_ptr.TweetID, true); err == nil {
					mxNotify(mxcli, "redaction", "Ok, I deleted that tweet for you")
				} else {
					log.Println("RedactTweetERROR:", err)
					mxNotify(mxcli, "redaction", "Could not redact your tweet")
				}
				if err := mclient.DeleteStatus(context.Background(), rums_ptr.TootID); err == nil {
					mxNotify(mxcli, "redaction", "Ok, I deleted that toot for you")
				} else {
					log.Println("RedactTweetERROR", err)
					mxNotify(mxcli, "redaction", "Could not redact your toot")
				}
			} else {
				mxNotify(mxcli, "redaction", "Won't redact other users status for you! Set admins_can_redact_user_status=true if you disagree.")
			}
		}()
	})

	/// Send a warning or welcome text to newly joined users
	if len(c.GetValueDefault("matrix", "join_welcome_text", "")) > 0 {
		syncer.OnEventType("m.room.member", func(ev *gomatrix.Event) {
			if mxIgnoreEvent(ev) { //ignore messages from ourselves or from other rooms in case of dual-login
				return
			}

			if membership, inmap := ev.Content["membership"]; inmap && membership == "join" {
				mxNotify(mxcli, "welcomer", c["matrix"]["join_welcome_text"])
			}
		})
	}

	///run event loop
	for {
		log.Println("syncing..")
		if err := mxcli.Sync(); err != nil {
			fmt.Println("Sync() returned ", err)
		}
		time.Sleep(100 * time.Second)
	}
}
