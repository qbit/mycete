package peanuts

type Links struct {
	Link        string `json:"link"`
	Text        string `json:"text"`
	Len         int    `json:"len"`
	Pos         int    `json:"pos"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Mentions struct {
	Id        string `json:"id"`
	Len       int    `json:"len"`
	Pos       int    `json:"pos"`
	Text      string `json:"text"`
	IsLeading bool   `json:"is_leading"`
	IsCopy    bool   `json:"is_copy"`
}

type Tags struct {
	Len  int    `json:"len"`
	Pos  int    `json:"pos"`
	Text string `json:"text"`
}

type Entities struct {
	Links    []Links    `json:"links"`
	Mentions []Mentions `json:"mentions"`
	Tags     []Tags     `json:"tags"`
}
