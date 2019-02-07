package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gokyle/goconfig"
	"github.com/matrix-org/gomatrix"
	"github.com/qbit/mycete/protector"
)

var c goconfig.ConfigMap
var err error
var temp_image_files_dir_ string

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

	if _, err := mxcli.JoinRoom(c["matrix"]["room_id"], "", nil); err != nil {
		panic(err)
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
							if c["server"]["mastodon"] == "true" {
								err = sendToot(mclient, post, ev.Sender)
								if err != nil {
									log.Println(err)
								}
								mxNotify(mxcli, "mastodon", "sent toot!")
							}

							if c["server"]["twitter"] == "true" {
								err = sendTweet(tclient, post, ev.Sender)
								if err != nil {
									log.Println(err)
								}
								mxNotify(mxcli, "twitter", "sent tweet!")
							}
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
				if urli, inmap := ev.Content["url"]; inmap {
					if url, ok := urli.(string); ok {
						go func() {
							lock := getPerUserLock(ev.Sender)
							lock.Lock()
							defer lock.Unlock()
							if err := saveMatrixFile(mxcli, ev.Sender, url); err != nil {
								mxNotify(mxcli, "error", "could not get your image")
								fmt.Println("ERROR downloading image", err)
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
	if c.GetValueDefault("images", "enabled", "false") == "true" {
		syncer.OnEventType("m.room.redaction", func(ev *gomatrix.Event) {
			if mxIgnoreEvent(ev) { //ignore messages from ourselves or from other rooms in case of dual-login
				return
			}
			go func() {
				lock := getPerUserLock(ev.Sender)
				lock.Lock()
				defer lock.Unlock()
				err := rmFile(ev.Sender)
				if err == nil || !os.IsNotExist(err) {
					mxNotify(mxcli, "redaction", fmt.Sprintf("%s's image has been redacted. Next toot/weet will not contain that image.", ev.Sender))
				}

			}()
		})
	}
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

	go mxRunBot()

	///wait until Signal
	{
		ctrlc_c := make(chan os.Signal, 1)
		signal.Notify(ctrlc_c, os.Interrupt, os.Kill, syscall.SIGTERM)
		<-ctrlc_c //block until ctrl+c is pressed || we receive SIGINT aka kill -1 || kill
		fmt.Println("Exiting")
	}
}
