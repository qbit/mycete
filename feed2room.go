package main

import (
	"context"
	"log"
	"strings"

	"github.com/matrix-org/gomatrix"
	mastodon "github.com/mattn/go-mastodon"
)

//// Desired Abilities (e.g.)
//// when acting as Account UserMe
//// 1. post all Status of UserMe to AdditionalMatrixRooms, no matter which client was used to write them (filtered Home timeline)
//// 2. post all Status of UserMe that DID NOT originate from controling matrix room to controlling matrix room
//// 3. post all Notifiations (private or not) to controlling matrix room (mentions, follows, etc)
//// 4. post all public mentions of UserMe to AdditionalMatrixRooms (filtered Home timeline)
//// 5. optionally post all Status with certain tag to AdditionalMatrixRooms (filtered HashTag timeline)

func (frc *FeedRoomConnector) uploadImageLinkToMatrix(imgurl string) MxUploadedImageInfo {
	future := make(chan MxUploadedImageInfo, 3)
	frc.mxlinkupload_c <- MxContentUrlFuture{imgurl: imgurl, future_mxcurl_c: future}
	return <-future
}

func (frc *FeedRoomConnector) writeNotificationToRoom(notification *mastodon.Notification, mroom string) {
	log.Println("writeNotificationToRoom:", mroom)
	text, htmltext := formatNotificationForMatrix(notification)
	frc.mxcli.SendMessageEvent(mroom, "m.room.message", gomatrix.HTMLMessage{MsgType: "m.notice", Format: "org.matrix.custom.html", Body: text, FormattedBody: htmltext})
}

func (frc *FeedRoomConnector) writeStatusToRoom(status *mastodon.Status, mroom string) {
	log.Println("writeStatusToRoom:", "status:", status.ID, "to room:", mroom)
	text, htmltext := formatStatusForMatrix(status)
	frc.mxcli.SendMessageEvent(mroom, "m.room.message", gomatrix.HTMLMessage{MsgType: "m.notice", Format: "org.matrix.custom.html", Body: text, FormattedBody: htmltext})

	if status.MediaAttachments != nil && len(status.MediaAttachments) > 0 && len(status.MediaAttachments) <= feed2matrx_image_count_limit_ {
		for _, attachment := range status.MediaAttachments {
			if attachment.Type == "image" || attachment.Type == "gifv" {
				imgurl := attachment.RemoteURL
				if len(imgurl) == 0 {
					imgurl = attachment.URL
				}
				if len(imgurl) == 0 {
					imgurl = attachment.PreviewURL
				}
				if len(imgurl) == 0 {
					continue // skip attachment since there is no URL
				}
				content_data := frc.uploadImageLinkToMatrix(imgurl)
				thumbnail_content_data := MxUploadedImageInfo{}
				if len(attachment.PreviewURL) > 0 {
					if attachment.PreviewURL != imgurl {
						thumbnail_content_data = frc.uploadImageLinkToMatrix(attachment.PreviewURL)
					} else {
						thumbnail_content_data = content_data
					}
				}
				if len(content_data.mxcurl) > 0 && content_data.err == nil {
					bodytext := attachment.Description
					if len(bodytext) == 0 {
						bodytext = status.URL
					}
					imgmsg := gomatrix.ImageMessage{
						MsgType: "m.image",
						Body:    bodytext,
						URL:     content_data.mxcurl,
						Info: gomatrix.ImageInfo{
							Height:   uint(attachment.Meta.Original.Height),
							Width:    uint(attachment.Meta.Original.Width),
							Mimetype: content_data.mimetype,
							Size:     uint(content_data.contentlength),
						},
					}
					if len(thumbnail_content_data.mxcurl) > 0 && thumbnail_content_data.err == nil {
						imgmsg.ThumbnailURL = thumbnail_content_data.mxcurl
						imgmsg.ThumbnailInfo = gomatrix.ImageInfo{
							Height:   uint(attachment.Meta.Small.Height),
							Width:    uint(attachment.Meta.Small.Width),
							Mimetype: thumbnail_content_data.mimetype,
							Size:     uint(thumbnail_content_data.contentlength),
						}
					}
					frc.mxcli.SendMessageEvent(mroom, "m.room.message", imgmsg)

				} else {
					log.Printf("writeStatusToRoom: Image not uploaded: attachment: %+v, imgurl: %s, Err: %s", attachment, imgurl, content_data.err)
				}
			}
		}
	}
}

func taskFilterMastodonStreamForRoom(frc *FeedRoomConnector, configname string, targetroomduplicatefilter chan<- *mastodon.Status, statusOut chan<- *mastodon.Status) (statusInRv chan<- *mastodon.Status) {
	//subconfiguration for additonal matrix rooms
	filter_reblogs := c.GetValueDefault(configname, "filter_reblogs", "false") == "true"
	filter_unfollowed := c.GetValueDefault(configname, "filter_unfollowed", "false") == "true"
	filter_sensitive := c.GetValueDefault(configname, "filter_sensitive", "false") == "true"
	filter_otherpeoplesposts := c.GetValueDefault(configname, "filter_otherpeoplesposts", "true") == "true"
	filter_myposts := c.GetValueDefault(configname, "filter_myposts", "true") == "true"
	filter_visibility := strings.Split(c.GetValueDefault(configname, "filter_visibility", ""), " ")
	if len(filter_visibility) == 1 && len(filter_visibility[0]) == 0 {
		filter_visibility = nil
	}
	filter_for_tags := strings.Split(c.GetValueDefault(configname, "filter_for_tags", ""), " ")
	if len(filter_for_tags) == 1 && len(filter_for_tags[0]) == 0 {
		filter_for_tags = nil
	}

	/// Filter Homestream for things to be sent to additional rooms
	//--> filter_ownposts_no_private_c		--> nil
	//										\-> filter_ownposts_duplicates_c
	return frc.taskPickStatusFromChannel(StatusFilterConfig{
		debugname:                  configname,
		must_have_one_of_tag_names: filter_for_tags,
		must_be_original:           filter_reblogs,
		must_be_unmuted:            true,
		must_not_be_sensitive:      filter_sensitive,
		check_visibility:           filter_visibility != nil,
		check_tagnames:             filter_for_tags != nil,
		must_have_visiblity:        filter_visibility,
		must_be_written_by_us:      filter_otherpeoplesposts,
		must_not_be_written_by_us:  filter_myposts,
		must_be_followed_by_us:     filter_unfollowed},
		targetroomduplicatefilter, statusOut)
}

func taskWriteMastodonBackIntoMatrixRooms(mclient *mastodon.Client, mxcli *gomatrix.Client) (markseen_rv chan<- mastodon.ID) {
	defer func() {
		if x := recover(); x != nil {
			log.Println(x)
			panic(x)
		}
	}()
	if mclient == nil || mxcli == nil {
		return // do nothing
	}

	frc := &FeedRoomConnector{
		mclient:        mclient,
		tclient:        nil,
		mxcli:          mxcli,
		mxlinkupload_c: taskUploadImageLinksToMatrix(mxcli),
	}

	//configuation for controlling room
	show_mastodon_notifications := c.GetValueDefault("feed2matrix", "show_mastodon_notifications", "true") == "true"
	show_own_toots_from_foreign_clients := c.GetValueDefault("feed2matrix", "show_own_toots_from_foreign_clients", "true") == "true"
	show_complete_home_stream := c.GetValueDefault("feed2matrix", "show_complete_home_stream", "false") == "true"

	//configuration for additonal matrix rooms
	configurations := strings.Split(c.GetValueDefault("feed2morerooms", "configurations", ""), " ")
	if len(configurations) == 1 && len(configurations[0]) == 0 {
		configurations = nil
	}
	subscribe_tagstreams := strings.Split(c.GetValueDefault("feed2morerooms", "subscribe_tagstreams", ""), " ")
	if len(subscribe_tagstreams) == 1 && len(subscribe_tagstreams[0]) == 0 {
		subscribe_tagstreams = nil
	}

	//set up duplicate filter for each target room as well as a goroutine for each target room
	room_duplicate_filter_targets := make(map[string]chan<- *mastodon.Status)
	var next_in_chain_ chan<- *mastodon.Status = nil
	for _, configname := range configurations {
		//join additional room
		target_room, trvexists := c.GetValue("feed2morerooms_"+configname, "target_room")
		if !trvexists {
			panic("target_room in [feed2morerooms_" + configname + "] is not set")
		}
		room_filter_c, inmap := room_duplicate_filter_targets[target_room]
		if !inmap {
			if target_room != c["matrix"]["room_id"] {
				log.Println("taskFilterMastodonStreamForRoom: joining room", target_room)
				if _, err := frc.mxcli.JoinRoom(target_room, "", nil); err != nil {
					panic(err)
				}
			}
			room_c := make(chan *mastodon.Status, 42)
			//--> filter_ownposts_duplicates_c	-->	nil
			//									\-> no_duplicate_status_c --> to additional rooms
			room_filter_c, _ = frc.taskFilterDuplicateStatus(target_room, room_c, nil)
			room_duplicate_filter_targets[target_room] = room_filter_c
			go func() {
				log.Println("writeMastodonFeedIntoAdditionalMatrixRooms: starting for", target_room)
				for status := range room_c {
					frc.writeStatusToRoom(status, target_room)
				}
			}()
		}
		next_in_chain_ = taskFilterMastodonStreamForRoom(frc, "feed2morerooms_"+configname, room_filter_c, next_in_chain_)
	}

	no_duplicate_or_selfsent_status_c := make(chan *mastodon.Status, 42)
	notification2myroom_c := make(chan *mastodon.Notification, 42)

	//--> filter_duplicates_and_selfsent_c	--> filter_ownposts_duplicates_c
	//										\-> no_duplicate_or_selfsent_status_c --> to controlling room
	filter_duplicates_and_selfsent_c, markseen_c := frc.taskFilterDuplicateStatus("controlroom", no_duplicate_or_selfsent_status_c, nil)

	/// Filter Homestream for things sent from our account but not from controlling channel
	//--> filter_ownposts_with_private_c		--> next_in_chain_
	//											\-> filter_duplicates_and_selfsent_c
	filter_ownposts_with_private_c := frc.taskPickStatusFromChannel(StatusFilterConfig{
		debugname:                  "controlroom",
		must_have_one_of_tag_names: nil,
		check_tagnames:             false,
		must_be_original:           false,
		must_be_unmuted:            true,
		must_not_be_sensitive:      false,
		check_visibility:           false,
		must_be_written_by_us:      !show_complete_home_stream,
		must_not_be_written_by_us:  false,
		must_be_followed_by_us:     false},
		filter_duplicates_and_selfsent_c, next_in_chain_)

	//subscribe home stream
	homestream, err := mclient.StreamingUser(context.Background())
	if err != nil {
		panic(err)
	}
	//--> homestream		--> filter_ownposts_c
	//						\-> notification2myroom_c
	go frc.runSplitMastodonEventStream(homestream, filter_ownposts_with_private_c, notification2myroom_c)

	//subscribe tags in addition to home stream
	for _, tag := range subscribe_tagstreams {
		log.Println("taskWriteMastodonBackIntoMatrixRooms: subscribing tag", tag)
		tagstream, err := mclient.StreamingHashtag(context.Background(), tag, false)
		if err != nil {
			panic(err)
		}
		//--> tagstream			--> next_in_chain_
		//						\-> nil
		go frc.runSplitMastodonEventStream(tagstream, next_in_chain_, nil)
	}

	//goroutine writing stuff to controlling room
	go func() {
		log.Println("writePublishedFeedsIntoControllingRoom: starting")
		for {
			select {
			case notification := <-notification2myroom_c:
				if show_own_toots_from_foreign_clients || show_complete_home_stream {
					frc.writeNotificationToRoom(notification, c["matrix"]["room_id"])
				}
			case foreignsentstatus := <-no_duplicate_or_selfsent_status_c:
				if show_mastodon_notifications {
					frc.writeStatusToRoom(foreignsentstatus, c["matrix"]["room_id"])
				}
			}
		}
	}()
	return markseen_c
}
