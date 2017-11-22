package peanuts

type ClientInfoResult struct {
	*CommonResponse
	Data ClientInfo `json:"data"`
}

// Get clinet
// https://pnut.io/docs/resources/clients#get-clients-id
func (c *Client) GetClient(id string) (result ClientInfosResult, err error) {
	response_ch := make(chan response)
	c.queryQueue <- query{url: CLIENT_API + "/" + id, data: &result, method: "GET", response_ch: response_ch}
	return result, (<-response_ch).err
}
