package peanuts

type Image struct {
	Link      string `json:"link"`
	IsDefault bool   `json:"is_default"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

type ContentOfUser struct {
	Text        string   `json:"text"`
	Html        string   `json:"html"`
	Entities    Entities `json:"entities"`
	AvatarImage Image    `json:"avatar_image"`
	CoverImage  Image    `json:"cover_image"`
}

type CountsOfUser struct {
	Bookmarks int `json:"bookmarks"`
	Clients   int `json:"clients"`
	Followers int `json:"followers"`
	Following int `json:"following"`
	Posts     int `json:"posts"`
	Users     int `json:"users"`
}

type Verified struct {
	Domain string `json:"domain"`
	Link   string `json:"link"`
}

type User struct {
	CreatedAt    string        `json:"created_at"`
	Guid         string        `json:"guid"`
	Id           string        `json:"id"`
	Locale       string        `json:"locale"`
	Timezone     string        `json:"timezon"`
	Type         string        `json:"type"`
	Username     string        `json:"username"`
	Name         string        `json:"name"`
	Content      ContentOfUser `json:"content"`
	Counts       CountsOfUser  `json:"counts"`
	FollowsYou   bool          `json:"follows_you"`
	YouBlocked   bool          `json:"you_blocked"`
	YouFollow    bool          `json:"you_follow"`
	YouMuted     bool          `json:"you_muted"`
	YouCanFollow bool          `json:"you_can_follow"`
	Verified     Verified      `json:"verified"`
}

type Source struct {
	Name string `json:"name"`
	Link string `json:"link"`
	Id   string `json:"id"`
}

type CountsOfPost struct {
	Bookmarks int `json:"bookmarks"`
	Replies   int `json:"replies"`
	Reposts   int `json:"reposts"`
	Threads   int `json:"threads"`
}

type ContentOfPost struct {
	Text           string   `json:"text"`
	Html           string   `json:"html"`
	Entities       Entities `json:"entities"`
	LinksNotParsed bool     `json:"links_not_parsed"`
}

type Post struct {
	CreatedAt     string        `json:"created_at"`
	Guid          string        `json:"guid"`
	Id            string        `json:"id"`
	IsDeleted     bool          `json:"is_deleted"`
	Source        Source        `json:"source"`
	User          User          `json:"user"`
	ThreadId      string        `json:"thread_id"`
	IsRevised     bool          `json:"is_revised"`
	Revision      string        `json:"revision"`
	ReplyTo       string        `json:"reply_to"`
	RepostOf      *Post         `json:"repost_of"`
	Counts        CountsOfPost  `json:"counts"`
	Content       ContentOfPost `json:"content"`
	YouBookmarked bool          `json:"you_bookmarked"`
	YouReposted   bool          `json:"you_reposted"`
	PaginationId  string        `json:"pagination_id"`
}

type Action struct {
	PaginationId string `json:"pagination_id"`
	EventDate    string `json:"event_date"`
	Action       string `json:"action"`
	Users        []User `json:"users"`
	Objects      []Post `json:"objects"`
}

type Presence struct {
	Id         string `json:"id"`
	LastSeenAt string `json:"last_seen_at"`
	Presence   string `json:"presence"`
}

type Full struct {
	Immutable bool     `json:"immutable"`
	You       bool     `json:"you"`
	UserIds   []string `json:"user_ids"`
}

type Write struct {
	*Full
	AnyUser bool `json:"any_user"`
}

type Read struct {
	*Write
	Public bool `json:"publicj"`
}

type Acl struct {
	Full  Full  `json:"full"`
	Write Write `json:"write"`
	Read  Read  `json:"read"`
}

type CountsOfChannel struct {
	Messages    int `json:"messages"`
	Subscribers int `json:"subscribers"`
}

type Channel struct {
	CreatedAt     string          `json:"created_at"`
	Id            string          `json:"id"`
	Type          string          `json:"type"`
	Owner         User            `json:"owner"`
	Acl           Acl             `json:"acl"`
	Counts        CountsOfChannel `json:"counts"`
	YouSubscribed bool            `json:"you_subscribed"`
	YouMuted      bool            `json:"you_muted"`
	HasUnread     bool            `json:"has_unread"`
	PaginationId  string          `json:"pagination_id"`
}

type CountsOfMessage struct {
	Replies int `json:"replies"`
}

type ContentOfMessage struct {
	Html     string   `json:"html"`
	Text     string   `json:"text"`
	Entities Entities `json:"entities"`
}

type Message struct {
	Id           string           `json:"id"`
	ChannelId    string           `json:"channel_id"`
	CreatedAt    string           `json:"created_at"`
	Source       Source           `json:"source"`
	IsDeleted    bool             `json:"is_deleted"`
	ThreadId     string           `json:"thread_id"`
	User         User             `json:"user"`
	Counts       CountsOfMessage  `json:"counts"`
	Content      ContentOfMessage `json:"content"`
	PaginationId string           `json:"pagination_id"`
}

type ContentOfClient struct {
	*ContentOfMessage
}

type ClientInfo struct {
	CreatedAt string          `json:"created_at"`
	CreatedBy User            `json:"created_by"`
	Id        string          `json:"id"`
	Link      string          `json:"link"`
	Name      string          `json:"name"`
	Posts     int             `json:"posts"`
	Content   ContentOfClient `json:"content"`
}

type Marker struct {
	Id         string `json:"id"`
	LastReadId string `json:"last_read_id"`
	Percentage int    `json:"percentage"`
	UpdatedAt  string `json:"updated_at"`
	Version    string `json:"version"`
	Name       string `json:"name"`
}
