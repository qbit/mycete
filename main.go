package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gokyle/goconfig"
	"github.com/matrix-org/gomatrix"
	mastodon "github.com/mattn/go-mastodon"
	"github.com/qbit/mycete/protector"
)

var c goconfig.ConfigMap
var err error
var temp_image_files_dir_ string

//// Desired Result / TODO
//// when acting as Account UserMe
//// 1. post all Status of UserMe to AdditionalMatrixRooms, no matter which client was used to write them (filtered Home timeline)
//// 2. post all Status of UserMe that DID NOT originate from controling matrix room to controlling matrix room
//// 3. post all Notifiations (private or not) to controlling matrix room (mentions, follows, etc)
//// 4. post all public mentions of UserMe to AdditionalMatrixRooms (filtered Home timeline)
//// 5. optionally post all Status with certain tag to AdditionalMatrixRooms (filtered HashTag timeline)

func writeMastodonBackIntoMatrixRooms(mclient *mastodon.Client, mxcli *gomatrix.Client) (markseen_rv chan<- mastodon.ID) {
	if mclient == nil || mxcli == nil {
		return // do nothing
	}

	//configuation for controlling room
	show_mastodon_notifications := c.GetValueDefault("matrix", "show_mastodon_notifications", "true") == "true"
	show_own_toots_from_foreign_clients := c.GetValueDefault("matrix", "show_own_toots_from_foreign_clients", "true") == "true"
	show_complete_home_stream := c.GetValueDefault("matrix", "show_complete_home_stream", "false") == "true"

	//configuration for additonal matrix rooms
	filter_reblogs := c.GetValueDefault("feed2matrix", "filter_reblogs", "false") == "true"
	filter_sensitive := c.GetValueDefault("feed2matrix", "filter_sensitive", "false") == "true"
	additional_target_room_ids := strings.Split(c.GetValueDefault("feed2matrix", "target_room_ids", ""), " ")
	filter_visibility := strings.Split(c.GetValueDefault("feed2matrix", "filter_visibility", ""), " ")
	subscribe_tagstreams := strings.Split(c.GetValueDefault("feed2matrix", "subscribe_tagstreams", ""), " ")
	filter_for_tags := strings.Split(c.GetValueDefault("feed2matrix", "filter_for_tags", ""), " ")

	//then join additional rooms
	for _, mroom := range additional_target_room_ids {
		if mroom != c["matrix"]["room_id"] {
			log.Println("writeMastodonBackIntoMatrixRooms: joining room", mroom)
			if _, err := mxcli.JoinRoom(mroom, "", nil); err != nil {
				panic(err)
			}
		}
	}

	no_duplicate_or_selfsent_status_c := make(chan *mastodon.Status, 42)
	no_duplicate_status_c := make(chan *mastodon.Status, 42)
	notification2myroom_c := make(chan *mastodon.Notification, 42)
	// for _, tag := range strings.Split(c.GetValueDefault("feed2matrix", "filter_tags", ""), " ") {
	// 	hashstream, err := mclient.StreamingHashtag(context.Background(), tag, true)
	// }

	frc := &FeedRoomConnector{
		mclient: mclient,
		tclient: nil,
		mxcli:   mxcli,
	}

	//--> filter_duplicates_and_selfsent_c	--> filter_ownposts_duplicates_c
	//										\-> no_duplicate_or_selfsent_status_c --> to controlling room
	filter_duplicates_and_selfsent_c, markseen_c := frc.filterDuplicateStatus(no_duplicate_or_selfsent_status_c, nil)

	//--> filter_ownposts_duplicates_c	-->	nil
	//									\-> no_duplicate_status_c --> to additional rooms
	filter_ownposts_duplicates_c, _ := frc.filterDuplicateStatus(no_duplicate_status_c, nil)

	//--> filter_ownposts_c		--> filter_tag_c
	//							\-> filter_ownposts_duplicates_c
	filter_ownposts_no_private_c := frc.filterAndHandleStatus(StatusFilterConfig{
		must_have_one_of_tag_names: nil,
		must_be_original:           false,
		must_be_unmuted:            true,
		must_not_be_sensitive:      true,
		check_visibility:           true,
		must_have_visiblity:        []string{"public"},
		must_be_written_by_us:      true,
		must_not_be_written_by_us:  false,
		must_be_followed_by_us:     false},
		filter_ownposts_duplicates_c, nil)

	//--> filter_ownposts_c		--> filter_tag_c
	//							\-> filter_ownposts_duplicates_c
	filter_ownposts_with_private_c := frc.filterAndHandleStatus(StatusFilterConfig{
		must_have_one_of_tag_names: nil,
		must_be_original:           false,
		must_be_unmuted:            true,
		must_not_be_sensitive:      false,
		check_visibility:           false,
		must_be_written_by_us:      true,
		must_not_be_written_by_us:  false,
		must_be_followed_by_us:     false},
		filter_duplicates_and_selfsent_c, filter_ownposts_no_private_c)

	//	{cloud}					--> userstream
	userstream, err := mclient.StreamingUser(context.Background())
	if err != nil {
		panic(err)
	}
	//--> userstream		--> filter_ownposts_c
	//						\-> notification2myroom_c
	go frc.goSplitMastodonEventStream(userstream, filter_ownposts_with_private_c, notification2myroom_c)

	/*
		//	{cloud}					--> tagstream
		tagstream, err := mclient.StreamingHashtag(context.Background(), "realraum", true)
		if err != nil {
			panic(err)
		}
		//--> tagstream			--> filter_tag_c
		//						\-> nil
			go frc.goSplitMastodonEventStream(tagstream, filter_tag_c, nil)
	*/

	//start writing mastodon feed messages to room
	if len(additional_target_room_ids) > 0 {
		go func() {
			log.Println("writeMastodonFeedIntoAdditionalMatrixRooms: starting")
			for status := range no_duplicate_status_c {
				for _, mroom := range additional_target_room_ids {
					log.Println("writeMastodonFeedIntoAdditionalMatrixRooms: sending notice to %s", mroom)
					mxcli.SendNotice(mroom, formatStatusForMatrix(status))
				}
			}
		}()
	}
	go func() {
		log.Println("writePublishedFeedsIntoControllingRoom: starting")
		for {
			select {
			case notification := <-notification2myroom_c:
				mxNotify(mxcli, "writePublishedFeedsIntoControllingRoom Notifcation", formatNotificationForMatrix(notification))
			case foreignsentstatus := <-no_duplicate_or_selfsent_status_c:
				mxNotify(mxcli, "writePublishedFeedsIntoControllingRoom Foreign Status", formatStatusForMatrix(foreignsentstatus))
			}
		}
	}()
	return markseen_c
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

func mxRunBot() {
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
		markseen_c = writeMastodonBackIntoMatrixRooms(mclient, mxcli)
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
					guard_prefix := c.GetValueDefault("matrix", "guard_prefix", "")
					if strings.HasPrefix(post, guard_prefix) {
						post = strings.TrimSpace(post[len(guard_prefix):])

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
			default:
				fmt.Printf("%s messages are currently not supported", mtype)
				//remove saved image file if present. We only attach an image once.
				if c.GetValueDefault("images", "enabled", "false") == "true" {
					go func() {
						lock := getPerUserLock(ev.Sender)
						lock.Lock()
						defer lock.Unlock()
						rmFile(ev.Sender)
					}()
				}
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

func main() {
	cfile := flag.String("conf", "/etc/mycete.conf", "Configuration file")
	flag.Parse()

	protector.Protect("stdio rpath cpath wpath fattr inet dns")

	c, err = goconfig.ParseFile(*cfile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if c.GetValueDefault("images", "enabled", "false") == "true" {
		temp_image_files_dir_, err = ioutil.TempDir(c.GetValueDefault("images", "temp_dir", "/tmp"), "mycete")
		if err != nil {
			panic(err)
		}
		if err = os.Chmod(temp_image_files_dir_, 0700); err != nil {
			panic(err)
		}
		defer os.RemoveAll(temp_image_files_dir_)
	}

	if c_charlimitstr, c_charlimitstr_set := c.GetValue("feed2matrix", "characterlimit"); c_charlimitstr_set && len(c_charlimitstr) > 0 {
		if charlimit, err := strconv.Atoi(c_charlimitstr); err == nil {
			matrix_notice_character_limit_ = charlimit
		}
	}

	go mxRunBot()

	///wait until Signal
	{
		ctrlc_c := make(chan os.Signal, 1)
		signal.Notify(ctrlc_c, os.Interrupt, os.Kill, syscall.SIGTERM)
		<-ctrlc_c //block until ctrl+c is pressed || we receive SIGINT aka kill -1 || kill
		fmt.Println("Exiting")
	}
}
