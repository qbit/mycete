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

const (
	twitter_net  string = "twitter"
	mastodon_net string = "mastodon"
)

var (
	mastodon_status_uri_re_ *regexp.Regexp
	twitter_status_uri_re_  *regexp.Regexp
	directmsg_re_           *regexp.Regexp
)

func init() {
	mastodon_status_uri_re_ = regexp.MustCompile(`^https?://[^/]+/@\w+/(\d+)$`)
	twitter_status_uri_re_ = regexp.MustCompile(`^https?://twitter\.com/.+/status(?:es)?/(\d+)$`)
	directmsg_re_ = regexp.MustCompile(`(?:^|\s)(@\w+(?:@[a-zA-Z0-9.]+)?)(?:\W|$)`)
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

type mastodon_action_cmd func(string) error
type twitter_action_cmd func(string) error

//TODO: accept strings in form:
// ✓ url (where we can detect twitter or mastodon)
// ✓ "toot <ID>" --> mastodon
// ✓ "status <ID>" --> mastdon
// ✓ "tweet <ID>" --> twitter
// ✓ "birdsite <ID>" --> twitter
// - last --> favourite the last received toot or tweet
func parseReblogFavouriteArgs(prefix, line string, mxcli *gomatrix.Client, mcmd mastodon_action_cmd, tcmd twitter_action_cmd) error {
	tort := ""
	statusidstr := ""
	args := strings.SplitN(strings.ToLower(strings.TrimSpace(line[len(prefix):])), " ", 3)
	if len(args) > 1 {
		switch args[0] {
		case "toot", "status":
			tort = mastodon_net
			statusidstr = args[1]
		case "tweet", "birdsite":
			tort = twitter_net
			statusidstr = args[1]
		}
	} else if len(args) == 1 {
		if args[0] == "last" {
			///TODO
			return fmt.Errorf("Sorry, 'last' not implemented yet")
		} else if matchlist := mastodon_status_uri_re_.FindStringSubmatch(args[0]); len(matchlist) >= 2 {
			tort = mastodon_net
			statusidstr = matchlist[1]
		} else if matchlist := twitter_status_uri_re_.FindStringSubmatch(args[0]); len(matchlist) >= 2 {
			tort = twitter_net
			statusidstr = matchlist[1]
		}
	}
	/// now execute
	switch tort {
	case twitter_net:
		return tcmd(statusidstr)
	case mastodon_net:
		return mcmd(statusidstr)
	default:
		return fmt.Errorf("Please say " + prefix + " followed by 'last', <status URL> or 'toot'/'tweet' <ID>")
	}
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
			switch mtype {
			case "m.text":
				if post, ok := ev.Body(); ok {
					log.Printf("Message: '%s'", post)
					if strings.HasPrefix(post, c["matrix"]["reblog_prefix"]) {
						/// CMD Reblogging

						go func() {
							if err := parseReblogFavouriteArgs(c["matrix"]["reblog_prefix"], post, mxcli,
								func(statusid string) error {
									_, err := mclient.Reblog(context.Background(), mastodon.ID(statusid))
									if err == nil {
										rums_store_chan <- RUMSStoreMsg{key: ev.ID, data: MsgStatusData{MatrixUser: ev.Sender, TootID: mastodon.ID(statusid), Action: actionReblog}}
									}

									return err
								},
								func(postidstr string) error {
									postid, err := strconv.ParseInt(postidstr, 10, 64)
									if err != nil {
										return err
									}
									if postid <= 0 {
										return fmt.Errorf("Sorry could not parse status id")
									}
									_, err = tclient.Retweet(postid, true)
									if err == nil {
										rums_store_chan <- RUMSStoreMsg{key: ev.ID, data: MsgStatusData{MatrixUser: ev.Sender, TweetID: postid, Action: actionReblog}}
									}

									return err
								},
							); err == nil {
								mxNotify(mxcli, "reblog", "Ok, I reblogged/retweeted that status for you")
							} else {
								mxNotify(mxcli, "reblog", fmt.Sprintf("error reblogging/retweeting: %s", err.Error()))
							}
						}()
					} else if strings.HasPrefix(post, c["matrix"]["favourite_prefix"]) {
						/// CMD Favourite

						go func() {
							err := parseReblogFavouriteArgs(c["matrix"]["favourite_prefix"], post, mxcli,
								func(statusid string) error {
									_, err := mclient.Favourite(context.Background(), mastodon.ID(statusid))
									if err == nil {
										rums_store_chan <- RUMSStoreMsg{key: ev.ID, data: MsgStatusData{MatrixUser: ev.Sender, TootID: mastodon.ID(statusid), Action: actionFav}}
									}
									return err
								},
								func(postidstr string) error {
									postid, err := strconv.ParseInt(postidstr, 10, 64)
									if err != nil {
										return err
									}
									if postid <= 0 {
										return fmt.Errorf("Sorry could not parse status id")
									}
									_, err = tclient.Favorite(postid)
									if err == nil {
										rums_store_chan <- RUMSStoreMsg{key: ev.ID, data: MsgStatusData{MatrixUser: ev.Sender, TweetID: postid, Action: actionFav}}
									}
									return err
								},
							)
							if err == nil {
								mxNotify(mxcli, "favourite", "Ok, I favourited that status for you")

							} else {
								mxNotify(mxcli, "favourite", fmt.Sprintf("error favouriting: %s", err.Error()))
							}
						}()
					} else if strings.HasPrefix(post, c["matrix"]["directtweet_prefix"]) {
						/// CMD Twitter Direct Message

						if c["server"]["twitter"] != "true" {
							return
						}

						post = strings.TrimSpace(post[len(c["matrix"]["directtweet_prefix"]):])

						if len(post) > character_limit_twitter_ {
							log.Println("Direct Tweet too long")
							mxNotify(mxcli, "directtweet", fmt.Sprintf("Not direct-tweeting this! Too long"))
							return
						}

						m := directmsg_re_.FindStringSubmatch(post)
						if len(m) < 2 {
							mxNotify(mxcli, "directtweet", "No can do! A direct message requires a recepient. Please mention an @screenname.")
							return
						}

						go func() {
							for _, rcpt := range m[1:] {
								err := sendTwitterDirectMessage(tclient, post, rcpt)
								if err != nil {
									mxNotify(mxcli, "directtweet", fmt.Sprintf("Error Twitter-direct-messaging %s: %s", rcpt, err.Error()))
								}
							}
						}()

					} else if strings.HasPrefix(post, c["matrix"]["directtoot_prefix"]) {
						/// CMD Mastodon Direct Toot

						log.Println("direct toot")

						if c["server"]["mastodon"] != "true" {
							return
						}

						post = strings.TrimSpace(post[len(c["matrix"]["directtoot_prefix"]):])

						if len(post) > character_limit_mastodon_ {
							log.Println("Direct Toot too long")
							mxNotify(mxcli, "directtoot", "Not tooting this! Too long")
							return
						}

						if directmsg_re_.MatchString(post) == false {
							mxNotify(mxcli, "directtoot", "No can do! A direct message requires a recepient. Please mention an @username.")
							return
						}

						go func() {
							lock := getPerUserLock(ev.Sender)
							lock.Lock()
							defer lock.Unlock()
							var reviewurl string
							var mastodonid mastodon.ID

							reviewurl, mastodonid, err = sendToot(mclient, post, ev.Sender, true)
							if markseen_c != nil {
								markseen_c <- mastodonid
							}
							if err != nil {
								log.Println("MastodonTootERROR:", err)
								mxNotify(mxcli, "mastodon", "ERROR while tooting!")
							} else {
								mxNotify(mxcli, "mastodon", fmt.Sprintf("sent direct toot! %s", reviewurl))
							}

							//remember posted status IDs
							rums_store_chan <- RUMSStoreMsg{key: ev.ID, data: MsgStatusData{ev.Sender, mastodonid, 0, actionPost}}

							//remove saved image file if present. We only attach an image once.
							if c.GetValueDefault("images", "enabled", "false") == "true" {
								rmAllUserFiles(ev.Sender)
							}

						}()

					} else if strings.HasPrefix(post, c["matrix"]["guard_prefix"]) {
						/// CMD Posting

						post = strings.TrimSpace(post[len(c["matrix"]["guard_prefix"]):])

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
								reviewurl, mastodonid, err = sendToot(mclient, post, ev.Sender, false)
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
							rums_store_chan <- RUMSStoreMsg{key: ev.ID, data: MsgStatusData{ev.Sender, mastodonid, twitterid, actionPost}}

							//remove saved image file if present. We only attach an image once.
							if c.GetValueDefault("images", "enabled", "false") == "true" {
								rmAllUserFiles(ev.Sender)
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
							if err := saveMatrixFile(mxcli, ev.Sender, ev.ID, url); err != nil {
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
				err := rmFile(ev.Sender, ev.Redacts)
				if err == nil {
					mxNotify(mxcli, "redaction", fmt.Sprintf("%s's image has been redacted. Next toot/weet will not contain that image.", ev.Sender))
				}
				if err != nil && !os.IsNotExist(err) {
					log.Println("ERROR deleting image:", err)
				}

			}()
		}
		go func() {
			future_chan := make(chan *MsgStatusData, 1)
			rums_retrieve_chan <- RUMSRetrieveMsg{key: ev.Redacts, future: future_chan}
			rums_ptr := <-future_chan
			if rums_ptr == nil {
				return
			}
			if c.GetValueDefault("matrix", "admins_can_redact_user_status", "false") == "true" || rums_ptr.MatrixUser == ev.Sender {
				switch rums_ptr.Action {
				case actionPost:
					if rums_ptr.TweetID > 0 {
						if _, err := tclient.DeleteTweet(rums_ptr.TweetID, true); err == nil {
							mxNotify(mxcli, "redaction", "Ok, I deleted that tweet for you")
						} else {
							log.Println("RedactTweetERROR:", err)
							mxNotify(mxcli, "redaction", "Could not redact your tweet")
						}
					}
					if len(rums_ptr.TootID) > 0 {
						if err := mclient.DeleteStatus(context.Background(), rums_ptr.TootID); err == nil {
							mxNotify(mxcli, "redaction", "Ok, I deleted that toot for you")
						} else {
							log.Println("RedactTweetERROR", err)
							mxNotify(mxcli, "redaction", "Could not redact your toot")
						}
					}
				case actionReblog:
					if rums_ptr.TweetID > 0 {
						if _, err := tclient.UnRetweet(rums_ptr.TweetID, true); err == nil {
							mxNotify(mxcli, "redaction", "Ok, I un-retweetet that tweet for you")
						} else {
							log.Println("RedactTweetERROR:", err)
							mxNotify(mxcli, "redaction", "Could not redact your retweet")
						}
					}
					if len(rums_ptr.TootID) > 0 {
						if _, err := mclient.Unreblog(context.Background(), rums_ptr.TootID); err == nil {
							mxNotify(mxcli, "redaction", "Ok, I un-reblogged that toot for you")
						} else {
							log.Println("RedactTweetERROR", err)
							mxNotify(mxcli, "redaction", "Could not redact your reblog")
						}
					}
				case actionFav:
					if rums_ptr.TweetID > 0 {
						if _, err := tclient.Unfavorite(rums_ptr.TweetID); err == nil {
							mxNotify(mxcli, "redaction", "Ok, I removed your favor from that tweet")
						} else {
							log.Println("RedactTweetERROR:", err)
							mxNotify(mxcli, "redaction", "Could not redact your favor")
						}
					}
					if len(rums_ptr.TootID) > 0 {
						if _, err := mclient.Unfavourite(context.Background(), rums_ptr.TootID); err == nil {
							mxNotify(mxcli, "redaction", "Ok, I removed your favour from that toot")
						} else {
							log.Println("RedactTweetERROR", err)
							mxNotify(mxcli, "redaction", "Could not redact your favour")
						}
					}

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
