package main

import (
	"context"
	"log"

	"github.com/ChimeraCoder/anaconda"
	"github.com/btittelbach/cachetable"
	"github.com/matrix-org/gomatrix"
	mastodon "github.com/mattn/go-mastodon"
)

type FeedRoomConnector struct {
	mclient        *mastodon.Client
	tclient        *anaconda.TwitterApi
	mxcli          *gomatrix.Client
	mxlinkupload_c chan<- MxContentUrlFuture
}

type StatusFilterConfig struct {
	debugname                  string
	must_be_written_by_us      bool
	must_not_be_written_by_us  bool
	check_visibility           bool
	must_have_visiblity        []string
	check_tagnames             bool
	must_have_one_of_tag_names []string
	must_be_unmuted            bool
	must_be_original           bool
	must_be_followed_by_us     bool
	must_not_be_sensitive      bool
}

func (frc *FeedRoomConnector) runSplitMastodonEventStream(evChan <-chan mastodon.Event, statusOutChan chan<- *mastodon.Status, notificationOutChan chan<- *mastodon.Notification) {
	for eventi := range evChan {
		switch event := eventi.(type) {
		case *mastodon.ErrorEvent:
			log.Println("runSplitMastodonEventStream:", "Error event: %s", event.Error())
			continue
		case *mastodon.UpdateEvent:
			if statusOutChan != nil {
				statusOutChan <- event.Status
			}
			// log.Println("runSplitMastodonEventStream: new Status", event.Status)
		case *mastodon.NotificationEvent:
			if notificationOutChan != nil {
				notificationOutChan <- event.Notification
			}
			// log.Println("runSplitMastodonEventStream: new Notification", event.Notification)
		case *mastodon.DeleteEvent:
			continue
		default:
			log.Println("runSplitMastodonEventStream:", "Unhandled event: %+v", eventi)
		}
	}
}

func (frc *FeedRoomConnector) taskJoinStatusStreams(statusOutChan chan<- *mastodon.Status) (statusOutChan1 <-chan *mastodon.Status, statusOutChan2 <-chan *mastodon.Status) {
	statusOutChan1 = make(chan *mastodon.Status, 42)
	statusOutChan2 = make(chan *mastodon.Status, 42)
	go func() {
		for {
			select {
			case s := <-statusOutChan1:
				statusOutChan <- s
			case s := <-statusOutChan2:
				statusOutChan <- s
			}
		}
	}()
	return
}

func (frc *FeedRoomConnector) taskPickStatusFromChannel(config StatusFilterConfig, statusPassedFilter chan<- *mastodon.Status, statusOut chan<- *mastodon.Status) (statusInRV chan<- *mastodon.Status) {
	statusIn := make(chan *mastodon.Status, 42)

	go func() {
		if statusOut != nil {
			defer close(statusOut)
		}
		my_account, err := frc.mclient.GetAccountCurrentUser(context.Background())
		if err != nil {
			panic(err)
		}
	FILTERFOR:
		for status := range statusIn {
			if statusOut != nil {
				//pass on copy to next handler
				select {
				case statusOut <- status:
				default:
				}
			}

			passes_tag_check := false
			passes_visibility_check := false
			passes_flag_check := !(status.Muted != nil && status.Muted.(bool) == true && config.must_be_unmuted) && !(status.Sensitive && config.must_not_be_sensitive) && !(config.must_be_original && ((status.Reblogged != nil && status.Reblogged.(bool) == true) || status.Reblog != nil))

			if !passes_flag_check {
				log.Println("taskPickStatusFromChannel:", config.debugname, status.ID, "failed flag check")
				continue FILTERFOR
			}

			if config.must_be_written_by_us && status.Account.ID != my_account.ID {
				log.Println("taskPickStatusFromChannel:", config.debugname, status.ID, "failed check: must be written by us BUT IS NOT")
				continue FILTERFOR
			}

			if config.must_not_be_written_by_us && status.Account.ID == my_account.ID {
				log.Println("taskPickStatusFromChannel:", config.debugname, status.ID, "failed check: must NOT be written by us BUT IS")
				continue FILTERFOR
			}

			if config.check_visibility && config.must_have_visiblity != nil && len(config.must_have_visiblity) > 0 {
				for _, visibilty_compare := range config.must_have_visiblity {
					if status.Visibility == visibilty_compare {
						passes_visibility_check = true
						break
					}
				}

				if !passes_visibility_check {
					log.Println("taskPickStatusFromChannel:", config.debugname, status.ID, "failed visibility check")
					continue FILTERFOR
				}
			}

			if config.must_be_followed_by_us {
				passes_follow_check := false
				if relationships, relerr := frc.mclient.GetAccountRelationships(context.Background(), []string{string(status.Account.ID)}); relerr == nil && len(relationships) > 0 {
					passes_follow_check = relationships[0].Following && !relationships[0].Blocking
				} else {
					log.Println("taskPickStatusFromChannel:", config.debugname, status.ID, ":FollowCheck: ", relerr)
					passes_follow_check = false
				}
				if !passes_follow_check {
					log.Println("taskPickStatusFromChannel:", config.debugname, status.ID, "failed follow check")
					continue FILTERFOR
				}
			}

			if config.check_tagnames && config.must_have_one_of_tag_names != nil && len(config.must_have_one_of_tag_names) > 0 {
			TAGFOR:
				for _, tag_compare := range config.must_have_one_of_tag_names {
					for _, tag := range status.Tags {
						if tag.Name == tag_compare {
							passes_tag_check = true
							break TAGFOR
						}
					}
				}
				if !passes_tag_check {
					log.Println("taskPickStatusFromChannel:", config.debugname, status.ID, " failed tag check")
					continue FILTERFOR
				}
			}

			//passed ALL check
			statusPassedFilter <- status
		}
	}()
	return statusIn
}

/// Filters out duplicate status i.e. only forwards each status once
/// uses a forgetful cachetable, so the number of remembered already seen status ids does not grow to infinity with infinite time
func (frc *FeedRoomConnector) taskFilterDuplicateStatus(debugname string, statusPassedFilter chan<- *mastodon.Status, statusOut chan<- *mastodon.Status) (statusInRv chan<- *mastodon.Status, markStatusSeenRv chan<- mastodon.ID) {
	statusIn := make(chan *mastodon.Status, 42)
	markStatusSeen := make(chan mastodon.ID, 42)
	go func() {
		defer close(statusOut)
		already_seen_map, err := cachetable.NewCacheTable(8, 3, true)
		if err != nil {
			panic(err)
		}
	FILTERFOR:
		for {
			select {

			case status, isopen := <-statusIn:
				if !isopen {
					return
				}
				if statusOut != nil {
					//passthrough
					statusOut <- status
				}
				if _, inmap := already_seen_map.Get(string(status.ID)); inmap {
					//already boosted this status "today", probably used more than one of our hashtags
					log.Println("taskFilterDuplicateStatus:", debugname, status.ID, "failed already seen check")
					continue FILTERFOR
				}

				already_seen_map.Set(string(status.ID), true)
				statusPassedFilter <- status
			case statusid := <-markStatusSeen:
				already_seen_map.Set(string(statusid), true)
			}
		}
	}()
	return statusIn, markStatusSeen
}
