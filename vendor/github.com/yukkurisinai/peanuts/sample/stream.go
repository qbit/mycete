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
	result, err := client.GlobalStream()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("====")
	for _, post := range result.Data {
		fmt.Println(post.User.Username, ":", post.Content.Text)
		fmt.Println("====")
	}
}
