package main

import (
	"fmt"
	"html"
	"regexp"
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

func sanitizeFormatStatusForMatrix(status *mastodon.Status) (url, body, htmlbody string) {
	tagstripper := bluemonday.NewPolicy()
	tagstripper.AllowElements("br")
	tagstripper_html := bluemonday.NewPolicy()
	tagstripper_html.AllowElements("br", "strike", "em", "i", "b", "strong", "code", "tt", "p")
	re_br2newline := regexp.MustCompile("<br[^/>]*/?>")

	htmlbody = tagstripper_html.Sanitize(status.Content)
	body = html.UnescapeString(strings.TrimSpace(re_br2newline.ReplaceAllString(tagstripper.Sanitize(status.Content), "\n")))
	url = status.URL

	if len(body) > matrix_notice_character_limit_ {
		body = body[0:matrix_notice_character_limit_] + "..."
	}

	return
}

func formatStatusForMatrix(status *mastodon.Status) (body, htmlbody string) {
	sender := formatUserNameForMatrix(status.Account)
	url, body, htmlbody := sanitizeFormatStatusForMatrix(status)

	body = fmt.Sprintf("%s [ %s ]>\n%s", sender, url, body)
	htmlbody = fmt.Sprintf("<u><strong>%s</strong> writes in <a href=\"%s\">%s</a>&gt;</u><br/>%s", sender, url, url, htmlbody)
	return
}

func formatNotificationForMatrix(notification *mastodon.Notification) (body, htmlbody string) {
	sender := formatUserNameForMatrix(notification.Account)
	var content_text string
	var content_html string
	var url string
	var visibility string
	if notification.Status != nil {
		url, content_text, content_html = sanitizeFormatStatusForMatrix(notification.Status)
		visibility = notification.Status.Visibility
	}
	switch notification.Type {
	case "mention":
		body = fmt.Sprintf("%s mentioned you in %s status [ %s ]:\n%s", sender, visibility, url, content_text)
		htmlbody = fmt.Sprintf("<u><strong>%s</strong> mentioned you in %s status <a href=\"%s\">%s</a>&gt;</u><br/>%s", sender, visibility, url, url, content_html)
	case "reblog":
		body = fmt.Sprintf("%s reblogged your status [ %s ]", sender, url)
		htmlbody = fmt.Sprintf("<strong>%s</strong> reblogged your status <a href=\"%s\">%s</a>", sender, url, url)
	case "favourite":
		body = fmt.Sprintf("%s favourited your status [ %s ]", sender, url)
		htmlbody = fmt.Sprintf("<strong>%s</strong> favourited your status <a href=\"%s\">%s</a>", sender, url, url)
	case "follow":
		body = fmt.Sprintf("%s is following you now", sender)
		htmlbody = fmt.Sprintf("<strong>%s</strong> is following you now", sender)
	case "follow_request":
		body = fmt.Sprintf("%s would like to follow you!", sender)
		htmlbody = fmt.Sprintf("<strong>%s</strong> would like to follow you!", sender)
	case "poll":
		body = fmt.Sprintf("the result of %s's poll is in: %s", sender, url)
		htmlbody = fmt.Sprintf("the result of <strong>%s</strong>'s poll is in: <a href=\"%s\">%s</a>", sender, url, url)
	default:
		body = fmt.Sprintf("received unsupported notification of type %s from %s", notification.Type, sender)
		htmlbody = fmt.Sprintf("received unsupported notification of type %s from %s", notification.Type, sender)
	}
	return
}
