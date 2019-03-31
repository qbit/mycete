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
	filter_otherpeoplesposts := c.GetValueDefault("feed2matrix", "filter_otherpeoplesposts", "true") == "true"
	additional_target_room_ids := strings.Split(c.GetValueDefault("feed2matrix", "target_room_ids", ""), " ")
	filter_visibility := strings.Split(c.GetValueDefault("feed2matrix", "filter_visibility", ""), " ")
	if len(filter_visibility) == 1 && len(filter_visibility[0]) == 0 {
		filter_visibility = nil
	}
	subscribe_tagstreams := strings.Split(c.GetValueDefault("feed2matrix", "subscribe_tagstreams", ""), " ")
	if len(subscribe_tagstreams) == 1 && len(subscribe_tagstreams[0]) == 0 {
		subscribe_tagstreams = nil
	}
	filter_for_tags := strings.Split(c.GetValueDefault("feed2matrix", "filter_for_tags", ""), " ")
	if len(filter_for_tags) == 1 && len(filter_for_tags[0]) == 0 {
		filter_for_tags = nil
	}

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
		must_have_one_of_tag_names: filter_for_tags,
		must_be_original:           filter_reblogs,
		must_be_unmuted:            true,
		must_not_be_sensitive:      filter_sensitive,
		check_visibility:           filter_visibility != nil,
		check_tagnames:             filter_for_tags != nil,
		must_have_visiblity:        filter_visibility,
		must_be_written_by_us:      filter_otherpeoplesposts,
		must_not_be_written_by_us:  false,
		must_be_followed_by_us:     false},
		filter_ownposts_duplicates_c, nil)

	//--> filter_ownposts_c		--> filter_tag_c
	//							\-> filter_ownposts_duplicates_c
	filter_ownposts_with_private_c := frc.filterAndHandleStatus(StatusFilterConfig{
		must_have_one_of_tag_names: nil,
		check_tagnames:             false,
		must_be_original:           false,
		must_be_unmuted:            true,
		must_not_be_sensitive:      false,
		check_visibility:           false,
		must_be_written_by_us:      !show_complete_home_stream,
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

	for _, tag := range subscribe_tagstreams {
		//	{cloud}					--> tagstream
		tagstream, err := mclient.StreamingHashtag(context.Background(), tag, true)
		if err != nil {
			panic(err)
		}
		//--> tagstream			--> filter_tag_c
		//						\-> nil
		go frc.goSplitMastodonEventStream(tagstream, filter_ownposts_duplicates_c, nil)
	}

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
				if show_own_toots_from_foreign_clients || show_complete_home_stream {
					mxNotify(mxcli, "writePublishedFeedsIntoControllingRoom Notifcation", formatNotificationForMatrix(notification))
				}
			case foreignsentstatus := <-no_duplicate_or_selfsent_status_c:
				if show_mastodon_notifications {
					mxNotify(mxcli, "writePublishedFeedsIntoControllingRoom Foreign Status", formatStatusForMatrix(foreignsentstatus))
				}
			}
		}
	}()
	return markseen_c
}
