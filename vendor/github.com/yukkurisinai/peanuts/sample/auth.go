package main

import (
	"fmt"
	"github.com/yukkurisinai/peanuts"
)

const (
	CLIENT_ID     = "client id"
	CLIENT_SECRET = "client secret"
	REDIRECT_URI  = "redirect uri"
)

func main() {
	client := peanuts.NewClient(CLIENT_ID, CLIENT_SECRET)
	fmt.Println(client.AuthURL(REDIRECT_URI, []string{"basic"}, "code"))
	fmt.Print("> ")

	var code string
	fmt.Scan(&code)

	result, err := client.AccessToken(code, REDIRECT_URI)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(result.AccessToken)
}
