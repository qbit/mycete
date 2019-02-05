package main

import "sync"

var user_mutex_map_ map[string]*sync.Mutex
var user_mutex_map_lock_ sync.Mutex

func init() {
	user_mutex_map_ = make(map[string]*sync.Mutex, 10)
}

func getPerUserLock(user string) *sync.Mutex {
	user_mutex_map_lock_.Lock()
	defer user_mutex_map_lock_.Unlock()
	if lock, inmap := user_mutex_map_[user]; inmap {
		return lock
	} else {
		lock := &sync.Mutex{}
		user_mutex_map_[user] = lock
		return lock
	}
}
