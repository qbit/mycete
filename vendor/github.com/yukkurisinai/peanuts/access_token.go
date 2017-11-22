package peanuts

import (
	"net/url"
	"strings"
)

type App struct {
	Id   string `json:"id"`
	Link string `json:"link"`
	Name string `json:"name"`
}

type Token struct {
	App      App      `json:"app"`
	Scopes   []string `json:"scopes"`
	User     User     `json:"user"`
	ClientId string   `json:"client_id"`
}

type AccessTokenResult struct {
	AccessToken string `json:"access_token"`
	Token       Token  `json:"token"`
	UserId      string `json:"user_id"`
	Username    string `json:"username"`
}

// Get AccessToken from authorization code
// https://pnut.io/docs/authentication/web-flows
func (c *Client) AccessToken(code string, redirectURI string) (result AccessTokenResult, err error) {
	v := url.Values{}
	v.Set("client_id", c.clientId)
	v.Set("client_secret", c.clientSecret)
	v.Set("code", code)
	v.Set("redirect_uri", redirectURI)
	v.Set("grant_type", "authorization_code")
	response_ch := make(chan response)
	c.queryQueue <- query{url: OAUTH_ACCESS_TOKEN_API, form: v, data: &result, method: "POST", response_ch: response_ch}
	return result, (<-response_ch).err
}

// Get AccessToken from password
// https://pnut.io/docs/authentication/password-flow
func (c *Client) AccessTokenFromPassword(username string, password string, scope []string) (result AccessTokenResult, err error) {
	v := url.Values{}
	v.Set("client_id", c.clientId)
	v.Set("password_grant_secret", c.passwordGrantSecret)
	v.Set("username", username)
	v.Set("password", password)
	v.Set("grant_type", "password")
	v.Set("scope", strings.Join(scope, ","))
	response_ch := make(chan response)
	c.queryQueue <- query{url: OAUTH_ACCESS_TOKEN_API, form: v, data: &result, method: "POST", response_ch: response_ch}
	return result, (<-response_ch).err
}
