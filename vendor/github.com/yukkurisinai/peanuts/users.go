package peanuts

import (
	"net/url"
	"strconv"
	"strings"
)

type UserResult struct {
	*CommonResponse
	Data User `json:"data"`
}

type UsersResult struct {
	*CommonResponse
	Data []User `json:"data"`
}

type StringResult struct {
	Data string `json:"data"`
}

type PresenceResult struct {
	*CommonResponse
	Data Presence `json:"data"`
}

type NumberResult struct {
	*CommonResponse
	Data int `json:"data"`
}

type MessagesResult struct {
	*CommonResponse
	Data []Message `json:"data"`
}

type ClientInfosResult struct {
	*CommonResponse
	Data []ClientInfo `json:"data"`
}

// Get user
// https://pnut.io/docs/resources/users/lookup#get-users-id
func (c *Client) GetUser(id string) (result UserResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get users
// https://pnut.io/docs/resources/users/lookup#get-users
func (c *Client) GetUsers(ids []string) (result UsersResult, err error) {
	v := url.Values{}
	v.Set("ids", strings.Join(ids, ","))

	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "?" + v.Encode(), data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Replace profile
// this func will be updated
// https://pnut.io/docs/resources/users/profile#put-users-me
func (c *Client) ReplaceProfile(json string) (result UserResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_ME_API, data: &result, method: "PUT", response_ch: response_ch, json: json}
	return result, (<-response_ch).err
}

// Update profile
// this func will be updated
// https://pnut.io/docs/resources/users/profile#patch-users-me
func (c *Client) UpdateProfile(json string) (result UserResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_ME_API, data: &result, method: "PATCH", response_ch: response_ch, json: json}
	return result, (<-response_ch).err
}

// Get avatar url
// https://pnut.io/docs/resources/users/profile#get-users-id-avatar
func (c *Client) GetAvatarURL(id string, v ...url.Values) (url string, err error) {
	param := ""
	if len(v) > 0 {
		param = v[0].Encode()
	}
	if param != "" {
		param = "?" + param
	}
	result := &StringResult{}
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/avatar" + param, data: &result, method: "GET", response_ch: response_ch, redirect: true}
	return result.Data, (<-response_ch).err
}

// Upload avatar
// this func will be updated
// https://pnut.io/docs/resources/users/profile#post-users-me-avatar
func (c *Client) UploadAvatar(avatar []byte) (result UserResult, err error) {
	return result, notSupported()
}

// Upload avatar from url
// this func will be updated
// https://pnut.io/docs/resources/users/profile#post-users-me-avatar
func (c *Client) UploadAvatarFromURL(url string) (result UserResult, err error) {
	return result, notSupported()
}

// Get avatar url
// https://pnut.io/docs/resources/users/profile#get-users-id-cover
func (c *Client) GetCoverURL(id string, v ...url.Values) (url string, err error) {
	param := ""
	if len(v) > 0 {
		param = v[0].Encode()
	}
	if param != "" {
		param = "?" + param
	}
	result := &StringResult{}
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/cover" + param, data: &result, method: "GET", response_ch: response_ch, redirect: true}
	return result.Data, (<-response_ch).err
}

// Upload cover
// this func will be updated
// https://pnut.io/docs/resources/users/profile#post-users-me-cover
func (c *Client) UploadCover(cover []byte) (result UserResult, err error) {
	return result, notSupported()
}

// Upload cover from url
// this func will be updated
// https://pnut.io/docs/resources/users/profile#post-users-me-cover
func (c *Client) UploadCoverFromURL(url string) (result UserResult, err error) {
	return result, notSupported()
}

// Get following
// https://pnut.io/docs/resources/users/following#get-users-id-following
func (c *Client) GetFollowing(id string, count ...int) (result UsersResult, err error) {
	v := url.Values{}
	if len(count) > 0 {
		v.Set("count", strconv.Itoa(count[0]))
	}
	param := v.Encode()
	if param != "" {
		param = "?" + param
	}
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/following" + param, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get followers
// https://pnut.io/docs/resources/users/followers#get-users-id-followers
func (c *Client) GetFollowers(id string, count ...int) (result UsersResult, err error) {
	v := url.Values{}
	if len(count) > 0 {
		v.Set("count", strconv.Itoa(count[0]))
	}
	param := v.Encode()
	if param != "" {
		param = "?" + param
	}
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/followers" + param, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Follow
// https://pnut.io/docs/resources/users/following#put-users-id-follow
func (c *Client) Follow(id string) (result UserResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/follow", data: &result, method: "PUT", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Delete follow
//https://pnut.io/docs/resources/users/following#delete-users-id-follow
func (c *Client) UnFollow(id string) (result UserResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/follow", data: &result, method: "DELETE", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get muted
// https://pnut.io/docs/resources/users/muting#get-users-id-muted
func (c *Client) GetMuted() (result UsersResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_ME_API + "/muted", data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Mute
// https://pnut.io/docs/resources/users/muteing#put-users-id-mute
func (c *Client) Mute(id string) (result UserResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/mute", data: &result, method: "PUT", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Delete mute
// https://pnut.io/docs/resources/users/muting#delete-users-id-mute
func (c *Client) UnMute(id string) (result UserResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/mute", data: &result, method: "DELETE", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get blocked
// https://pnut.io/docs/resources/users/muting#get-users-id-blocked
func (c *Client) GetBlocked() (result UsersResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_ME_API + "/blocked", data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Block
// https://pnut.io/docs/resources/users/blocking#put-users-id-block
func (c *Client) Block(id string) (result UserResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/block", data: &result, method: "PUT", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Delete block
// https://pnut.io/docs/resources/posts/bookmarks#delete-posts-id-bookmark
func (c *Client) UnBlock(id string) (result UserResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/block", data: &result, method: "DELETE", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get presence
// https://pnut.io/docs/resources/users/presence#get-users-id-presence
func (c *Client) GetPresence(id string) (result PresenceResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/presence", data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Update presence
// https://pnut.io/docs/resources/users/presence#put-users-id-presence
func (c *Client) SetPresence(presence string) (result PresenceResult, err error) {
	v := url.Values{}
	v.Set("presence", presence)
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/me/presence", form: v, data: &result, method: "PUT", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get mentions
// https://pnut.io/docs/resources/posts/streams#get-users-id-mentions
func (c *Client) GetMentions(id string, count ...int) (result PostsResult, err error) {
	v := url.Values{}
	if len(count) > 0 {
		v.Set("count", strconv.Itoa(count[0]))
	}
	param := v.Encode()
	if param != "" {
		param = "?" + param
	}
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/mentions" + param, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get posts from id
// https://pnut.io/docs/resources/posts/streams#get-users-id-posts
func (c *Client) GetPostsFromUser(id string, count ...int) (result PostsResult, err error) {
	v := url.Values{}
	if len(count) > 0 {
		v.Set("count", strconv.Itoa(count[0]))
	}
	param := v.Encode()
	if param != "" {
		param = "?" + param
	}
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/posts" + param, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get bookmarks
// https://pnut.io/docs/resources/posts/bookmarks#get-users-id-bookmarks
func (c *Client) GetBookmarks(id string, count ...int) (result PostsResult, err error) {
	v := url.Values{}
	if len(count) > 0 {
		v.Set("count", strconv.Itoa(count[0]))
	}
	param := v.Encode()
	if param != "" {
		param = "?" + param
	}
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/bookmarks" + param, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get actions for me
// https://pnut.io/docs/resources/posts/actions#get-posts-id-actions
func (c *Client) GetActionsForMe(v ...url.Values) (result ActionsResult, err error) {
	param := ""
	if len(v) > 0 {
		param = v[0].Encode()
	}
	if param != "" {
		param = "?" + param
	}
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_ME_API + "/actions" + param, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get channels by me
// https://pnut.io/docs/resources/channels/lookup#get-users-me-channels
func (c *Client) GetChannelsByMe() (result ChannelsResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_ME_API + "/channels", data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get number of unread private message
// https://pnut.io/docs/resources/channels/lookup#get-users-me-channels-num_unread-pm
func (c *Client) GetNumberOfUnreadPM() (result NumberResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: UNREAD_PM_NUMBER_API, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Mark read all private message
// https://pnut.io/docs/resources/channels/lookup#delete-users-me-channels-num_unread-pm
func (c *Client) MarkReadPM() (result NumberResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: UNREAD_PM_NUMBER_API, data: &result, method: "DELETE", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get subscribed channels
// https://pnut.io/docs/resources/channels/subscribing#get-users-me-channels-subscribed
func (c *Client) GetSubscribedChannels() (result ChannelsResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: SUBSCRIBED_CHANNELS_API, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get muted channels
// https://pnut.io/docs/resources/channels/muting#get-users-me-channels-muted
func (c *Client) GetMutedChannels() (result ChannelsResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: MUTED_CHANNELS_API, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get messages by me
// https://pnut.io/docs/resources/messages/lookup#get-users-me-messages
func (c *Client) GetMessagesByMe() (result MessagesResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: ME_MESSAGES_API, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get client infos
// https://pnut.io/docs/resources/clients#get-users-id-clients
func (c *Client) GetClients(id string) (result ClientInfosResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: USER_API + "/" + id + "/clients", data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}
