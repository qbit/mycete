package main

import (
	"fmt"
	"github.com/yukkurisinai/peanuts"
)

const (
	CLIENT_ID     = "client id"
	CLIENT_SECRET = "client secret"
	ACCESS_TOKEN  = "access token"
)

func main() {
	client := peanuts.NewClient(CLIENT_ID, CLIENT_SECRET)
	client.SetAccessToken(ACCESS_TOKEN)
	v := url.Values{}
	v.Set("text", "Hello pnut.io")
	_, err := client.Post(v)
	if err != nil {
		fmt.Println(err)
		return
	}
}
