package peanuts

type MarkersResult struct {
	*CommonResponse
	Data []Marker `json:"data"`
}

// Set marker
// this func will be updated
// https://pnut.io/docs/resources/stream-marker
// https://pnut.io/docs/resources/stream-marker#post-markers
func (c *Client) SetMarker(json string) (result MarkersResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: MARKER_API, data: &result, method: "PUT", response_ch: response_ch, json: json}
	return result, (<-response_ch).err
}
