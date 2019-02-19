package main

import mastodon "github.com/mattn/go-mastodon"

type MsgStatusTripple struct {
	MatrixUser string
	TootID     mastodon.ID
	TweetID    int64
}

type RUMSStoreMsg struct {
	key  string
	data MsgStatusTripple
}

type RUMSRetrieveMsg struct {
	key    string
	future chan<- *MsgStatusTripple
}

func runRememberUsersMessageToStatus() (rv_store_chan chan<- RUMSStoreMsg, rv_retrieve_chan chan<- RUMSRetrieveMsg) {
	store_chan := make(chan RUMSStoreMsg, 20)
	retrieve_chan := make(chan RUMSRetrieveMsg, 20)
	go func() {
		brain := make(map[string]MsgStatusTripple, 100)
		for {
			select {
			case storeme, chanok := <-store_chan:
				if !chanok {
					return
				}
				brain[storeme.key] = storeme.data
			case retrieveme, chanok := <-retrieve_chan:
				if !chanok {
					return
				}
				rums, inmap := brain[retrieveme.key]
				if inmap {
					retrieveme.future <- &rums //return pointer to copy produced by map retrieval
				} else {
					retrieveme.future <- nil
				}
			}
		}
	}()
	return store_chan, retrieve_chan
}
