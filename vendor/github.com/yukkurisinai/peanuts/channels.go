package peanuts

import (
	"encoding/json"
	"net/url"
	"strings"
)

type ChannelResult struct {
	*CommonResponse
	Data Channel `json:"data"`
}

type ChannelsResult struct {
	*CommonResponse
	Data []Channel `json:"data"`
}

type createChannel struct {
	Type string `json:"type"`
	Acl  Acl    `json:"acl"`
}

// Get channel
// https://pnut.io/docs/resources/channels/lookup#get-channels-id
func (c *Client) GetChannel(id string) (result ChannelResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + id, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get channels
// https://pnut.io/docs/resources/channels/lookup#get-channels
func (c *Client) GetChannels(ids []string) (result UsersResult, err error) {
	v := url.Values{}
	v.Set("ids", strings.Join(ids, ","))

	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "?" + v.Encode(), data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Create channel
// https://pnut.io/docs/resources/channels/lifecycle#post-channels
func (c *Client) CreateChannel(typeStr string, acl Acl) (result ChannelResult, err error) {
	json, err := json.Marshal(&createChannel{Type: typeStr, Acl: acl})
	if err != nil {
		return
	}
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API, data: &result, method: "POST", response_ch: response_ch, json: string(json)}
	return result, (<-response_ch).err
}

// Update channel
// https://pnut.io/docs/resources/channels/lifecycle#put-channels-id
func (c *Client) UpdateChannel(id string, acl Acl) (result ChannelResult, err error) {
	json, err := json.Marshal(&createChannel{Acl: acl})
	if err != nil {
		return
	}
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + id, data: &result, method: "PUT", response_ch: response_ch, json: string(json)}
	return result, (<-response_ch).err
}

// Delete channel
// https://pnut.io/docs/resources/channels/lifecycle#delete-channels-id
func (c *Client) DeleteChannel(id string) (result ChannelResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + id, data: &result, method: "DELETE", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get subscribers of channel
// https://pnut.io/docs/resources/channels/subscribing#get-channels-id-subscribers
func (c *Client) GetSubscribersOfChannel(id string) (result UsersResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + id + "/subscribers", data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Subscribe channel
// https://pnut.io/docs/resources/channels/subscribing#put-channels-id-subscribe
func (c *Client) SubscribeChannel(id string) (result ChannelResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + id + "/subscribe", data: &result, method: "PUT", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Delete subscribe
// https://pnut.io/docs/resources/channels/subscribing#delete-channels-id-subscribe
func (c *Client) UnSubscribeChannel(id string) (result ChannelResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + id + "/subscribe", data: &result, method: "DELETE", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Mute channel
// https://pnut.io/docs/resources/channels/muting#put-channels-id-mute
func (c *Client) MuteChannel(id string) (result ChannelResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + id + "/mute", data: &result, method: "PUT", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Delete mute
// https://pnut.io/docs/resources/channels/muting#delete-channels-id-mute
func (c *Client) UnMuteChannel(id string) (result ChannelResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + id + "/mute", data: &result, method: "DELETE", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get message
// https://pnut.io/docs/resources/messages/lookup#get-channels-id-messages-id
func (c *Client) GetMessage(channelId string, messageId string) (result PostResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + channelId + "/messages/" + messageId, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get messages in thread
// https://pnut.io/docs/resources/messages/lookup#get-channels-id-messages-id-thread
func (c *Client) GetMessagesInThread(channelId string, messageId string) (result PostsResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + channelId + "/messages/" + messageId + "/thread", data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get messages from ids
// https://pnut.io/docs/resources/messages/lookup#get-messages
func (c *Client) GetMessages(ids []string) (result PostsResult, err error) {
	v := url.Values{}
	v.Set("ids", strings.Join(ids, ","))

	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/messages?" + v.Encode(), data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get messages of channel
// https://pnut.io/docs/resources/messages/lookup#get-channels-id-messages
func (c *Client) GetMessagesOfChannel(id string) (result PostsResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + id + "/messages", data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Post message
// https://pnut.io/docs/resources/messages/lifecycle#post-channels-id-messages
func (c *Client) CreateMessage(id string, text string) (result PostResult, err error) {
	v := url.Values{}
	v.Set("text", text)

	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + id + "/messages", form: v, data: &result, method: "POST", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Delete message
// https://pnut.io/docs/resources/messages/lifecycle#delete-channels-id-messages-id
func (c *Client) DeleteMessage(channelId string, messageId string) (result PostResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + channelId + "/messages/" + messageId, data: &result, method: "DELETE", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get sticky messages
// https://pnut.io/docs/resources/messages/sticky#get-channels-id-sticky_messages
func (c *Client) GetStickyMessages(id string) (result PostsResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + id + "/sticky_messages", data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Sticky message
// https://pnut.io/docs/resources/messages/sticky#put-channels-id-messages-id-sticky
func (c *Client) Sticky(channelId string, messageId string) (result PostResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + channelId + "/messages/" + messageId + "/sticky", data: &result, method: "PUT", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Delete sticky
// https://pnut.io/docs/resources/messages/sticky#delete-channels-id-messages-id-sticky
func (c *Client) UnSticky(channelId string, messageId string) (result PostResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CHANNEL_API + "/" + channelId + "/messages/" + messageId + "/sticky", data: &result, method: "DELETE", response_ch: response_ch}
	return result, (<-response_ch).err
}
