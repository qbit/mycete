package peanuts

import (
	"net/url"
	"strconv"
	"strings"
)

type PostResult struct {
	*CommonResponse
	Data Post `json:"data"`
}

type PostsResult struct {
	*CommonResponse
	Data []Post `json:"data"`
}

type ActionsResult struct {
	*CommonResponse
	Data []Action `json:"data"`
}

// Get post
// https://pnut.io/docs/resources/posts/lookup#get-posts-id
func (c *Client) GetPost(id string) (result PostResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: POST_API + "/" + id, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get posts
// https://pnut.io/docs/resources/posts/lookup#get-posts
func (c *Client) GetPosts(ids []string) (result PostsResult, err error) {
	v := url.Values{}
	v.Set("ids", strings.Join(ids, ","))
	response_ch := make(chan response)
	c.queryQueue <- query{url: POST_API + "?" + v.Encode(), data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get list of previous versions of a post
// https://pnut.io/docs/resources/posts/lookup#get-posts-id-revisions
func (c *Client) GetPostRevisions(id string) (result PostsResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: POST_API + "/" + id + "/revisions", data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Post post
// https://pnut.io/docs/resources/posts/lifecycle#post-posts
func (c *Client) Post(v url.Values) (result PostResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: POST_API, form: v, data: &result, method: "POST", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Revise post
// https://pnut.io/docs/resources/posts/lifecycle#put-posts-id
func (c *Client) RevisePost(id string, v url.Values) (result PostResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: POST_API + "/" + id, form: v, data: &result, method: "PUT", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Delete post
// https://pnut.io/docs/resources/posts/lifecycle#delete-posts-id
func (c *Client) DeletePost(id string) (result PostResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: POST_API + "/" + id, data: &result, method: "DELETE", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Me stream
// https://pnut.io/docs/resources/posts/streams#get-posts-streams-me
func (c *Client) MeStream(count ...int) (result PostsResult, err error) {
	v := url.Values{}
	if len(count) > 0 {
		v.Set("count", strconv.Itoa(count[0]))
	}
	param := v.Encode()
	if param != "" {
		param = "?" + param
	}
	response_ch := make(chan response)
	c.queryQueue <- query{url: STREAM_ME_API + param, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Unified stream
// https://pnut.io/docs/resources/posts/streams#get-posts-streams-unified
func (c *Client) UnifiedStream(count ...int) (result PostsResult, err error) {
	v := url.Values{}
	if len(count) > 0 {
		v.Set("count", strconv.Itoa(count[0]))
	}
	param := v.Encode()
	if param != "" {
		param = "?" + param
	}
	response_ch := make(chan response)
	c.queryQueue <- query{url: STREAM_UNIFIED_API + param, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Global stream
// https://pnut.io/docs/resources/posts/streams#get-posts-streams-unified
func (c *Client) GlobalStream(count ...int) (result PostsResult, err error) {
	v := url.Values{}
	if len(count) > 0 {
		v.Set("count", strconv.Itoa(count[0]))
	}
	param := v.Encode()
	if param != "" {
		param = "?" + param
	}
	response_ch := make(chan response)
	c.queryQueue <- query{url: STREAM_GLOBAL_API + param, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Tag stream
// https://pnut.io/docs/resources/posts/streams#get-posts-tag-tag
func (c *Client) TagStream(tag string, count ...int) (result PostsResult, err error) {
	v := url.Values{}
	if len(count) > 0 {
		v.Set("count", strconv.Itoa(count[0]))
	}
	param := v.Encode()
	if param != "" {
		param = "?" + param
	}
	response_ch := make(chan response)
	c.queryQueue <- query{url: STREAM_TAG_BASE_URL + "/" + tag + param, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Retrieve posts within thread
// https://pnut.io/docs/resources/posts/threads#get-posts-id-thread
func (c *Client) GetThread(id string) (result PostsResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: POST_API + "/" + id + "/thread", data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Repost post
// https://pnut.io/docs/resources/posts/reposts#put-posts-id-repost
func (c *Client) Repost(id string) (result PostResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: POST_API + "/" + id + "/repost", data: &result, method: "PUT", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Delete repost
// https://pnut.io/docs/resources/posts/reposts#delete-posts-id-repost
func (c *Client) UnRepost(id string) (result PostResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: POST_API + "/" + id + "/repost", data: &result, method: "DELETE", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Bookmark post
// https://pnut.io/docs/resources/posts/bookmarks#put-posts-id-bookmark
func (c *Client) Bookmark(id string) (result PostResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: POST_API + "/" + id + "/bookmark", data: &result, method: "PUT", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Delete bookmark
// https://pnut.io/docs/resources/posts/bookmarks#put-posts-id-bookmark
func (c *Client) UnBookmark(id string) (result PostResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: POST_API + "/" + id + "/bookmark", data: &result, method: "DELETE", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get actions
// https://pnut.io/docs/resources/posts/actions#get-posts-id-actions
func (c *Client) GetActions(id string, v ...url.Values) (result ActionsResult, err error) {
	param := ""
	if len(v) > 0 {
		param = v[0].Encode()
	}
	if param != "" {
		param = "?" + param
	}
	response_ch := make(chan response)
	c.queryQueue <- query{url: POST_API + "/" + id + "/actions" + param, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}
