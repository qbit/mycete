package peanuts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Client struct {
	clientId            string
	clientSecret        string
	passwordGrantSecret string
	queryQueue          chan query
	Api                 Api
}

func NewClient(clientId string, clientSecret string) *Client {
	queue := make(chan query)
	client := &Client{clientId: clientId, clientSecret: clientSecret, queryQueue: queue}
	client.initialize()
	go client.throttledQuery()
	return client
}

type query struct {
	url         string
	form        url.Values
	data        interface{}
	method      string
	response_ch chan response
	json        string
	redirect    bool
}

type response struct {
	data interface{}
	err  error
}

func (c *Client) initialize() {
	c.Api = *&Api{
		accessToken: "",
		HttpClient:  http.DefaultClient,
	}
}

// Generate authorization url
// https://pnut.io/docs/authentication/web-flows
func (c *Client) AuthURL(redirectURI string, scope []string, responseType string) string {
	return AUTHENTICATE_URL + "?client_id=" + c.clientId + "&redirect_uri=" + redirectURI + "&scope=" + strings.Join(scope, "%20") + "&response_type=" + responseType
}

// Set password grant secret
// https://pnut.io/docs/authentication/password-flow
func (c *Client) SetPasswordGrantSecret(passwordGrantSecret string) {
	c.passwordGrantSecret = passwordGrantSecret
}

// Set access token
// https://pnut.io/docs/authentication/web-flows
// https://pnut.io/docs/authentication/password-flow
func (c *Client) SetAccessToken(accessToken string) {
	c.Api.accessToken = accessToken
}

type StreamMeta struct {
	More  bool   `json:"more"`
	MaxId string `json:"max_id"`
	MinId string `json:"min_id"`
}

type Meta struct {
	*StreamMeta
	Code         int    `json:"code"`
	Error        string `json:"error"`
	ErrorMessage string `json:"error_message"`
}

type CommonResponse struct {
	Meta Meta `json:"meta"`
}

func decodeResponse(res *http.Response, data interface{}) error {
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, data)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		common := &CommonResponse{}
		err = json.Unmarshal(b, common)
		if err != nil {
			return err
		}
		return fmt.Errorf(strconv.Itoa(res.StatusCode) + ": " + common.Meta.ErrorMessage)
	}

	return nil
}

func (c *Client) execQuery(url string, form url.Values, data interface{}, method string, jsonStr string, redirect bool) (err error) {
	var req *http.Request
	if jsonStr == "" {
		req, err = http.NewRequest(
			method,
			url,
			strings.NewReader(form.Encode()),
		)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequest(
			method,
			url,
			bytes.NewBuffer([]byte(jsonStr)),
		)
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		return err
	}
	if c.Api.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.Api.accessToken)
	}
	if redirect {
		res, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		if res.StatusCode == 301 || res.StatusCode == 302 || res.StatusCode == 303 || res.StatusCode == 307 {
			if len(res.Header["Location"]) > 0 {
				// gross
				// must fix with reflect
				err = json.Unmarshal([]byte("{\"data\":\""+res.Header["Location"][0]+"\"}"), data)
				return err
			} else {
				return fmt.Errorf("location is not found from header")
			}
		} else {
			return fmt.Errorf(strconv.Itoa(res.StatusCode))
		}
	}
	res, err := c.Api.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return decodeResponse(res, data)
}

func (c *Client) throttledQuery() {
	for q := range c.queryQueue {
		url := q.url
		form := q.form
		data := q.data
		method := q.method
		jsonStr := q.json
		redirect := q.redirect

		response_ch := q.response_ch

		err := c.execQuery(url, form, data, method, jsonStr, redirect)

		response_ch <- response{data, err}
	}
}

func notSupported() error {
	return fmt.Errorf("not supported")
}
