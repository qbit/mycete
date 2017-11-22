package peanuts

type PresencesResult struct {
	*CommonResponse
	Data []Presence `json:"data"`
}

// Get presences
// https://pnut.io/docs/resources/users/presence#get-presence
func (c *Client) GetPresences() (result PresencesResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: PRESENCE_API, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}
