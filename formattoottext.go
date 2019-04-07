package main

import (
	"fmt"
	"html"
	"strings"

	mastodon "github.com/mattn/go-mastodon"
	"github.com/microcosm-cc/bluemonday"
)

func formatUserNameForMatrix(acct mastodon.Account) string {
	tagstripper := bluemonday.StrictPolicy()
	if sender3 := strings.TrimSpace(html.UnescapeString(tagstripper.Sanitize(acct.DisplayName))); len(sender3) > 0 {
		return sender3
	}
	if sender2 := strings.TrimSpace(html.UnescapeString(tagstripper.Sanitize(acct.Username))); len(sender2) > 0 {
		return sender2
	}
	sender1 := strings.TrimSpace(html.UnescapeString(tagstripper.Sanitize(acct.Acct)))
	return sender1
}

func formatStatusForMatrix(status *mastodon.Status) (body, htmlbody string) {
	// sender:=status.Account.Username
	// sender:=status.Account.Acct
	tagstripper := bluemonday.NewPolicy()
	tagstripper.AllowElements("br")

	sender := formatUserNameForMatrix(status.Account)
	htmlbody = tagstripper.Sanitize(status.Content)
	body = html.UnescapeString(strings.TrimSpace(strings.ReplaceAll(htmlbody, "<br/>", "\n")))
	url := status.URL

	if len(body) > matrix_notice_character_limit_ {
		body = body[0:matrix_notice_character_limit_] + "..."
	}

	body = fmt.Sprintf("%s> %s\n%s", sender, body, url)
	htmlbody = fmt.Sprintf("<u>%s</u>&gt;<br/><p>%s</p><a href=\"%s\" style=\"font-size:80%%;\">%s</a>", sender, htmlbody, url, url)
	return
}

func formatNotificationForMatrix(notification *mastodon.Notification) string {
	// sender:=status.Account.Username
	// sender:=status.Account.Acct
	tagstripper := bluemonday.NewPolicy()
	tagstripper.AllowElements("br")
	sender := formatUserNameForMatrix(notification.Account)
	contenttext := ""
	url := ""
	if notification.Status != nil {
		url = notification.Status.URL
		contenttext = tagstripper.Sanitize(notification.Status.Content)
		contenttext = strings.ReplaceAll(contenttext, "<br/>", "\n")
	}
	switch notification.Type {
	case "mention":
		return fmt.Sprintf("%s mentioned you:\n%s\n%s", sender, contenttext, url)
	case "reblog":
		return fmt.Sprintf("%s reblogged your Status\n%s", sender, url)
	case "favourite":
		return fmt.Sprintf("%s favourited your Status\n%s", sender, url)
	case "follow":
		return fmt.Sprintf("%s is following you now", sender)
	}
	return ""
}
