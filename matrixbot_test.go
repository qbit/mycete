package main

import (
	"regexp"
	"testing"
)

func testMatchRegex(t *testing.T, name string, re *regexp.Regexp, mustmatch, mustnotmatch []string) {
	for _, str := range mustmatch {
		if re.MatchString(str) == false {
			t.Errorf("regexp %s did not match test string: %s", name, str)
		}
	}
	for _, str := range mustnotmatch {
		if re.MatchString(str) == true {
			t.Errorf("regexp %s did match negative test string: %s", name, str)
		}
	}
}

func testSubmatchRegex(t *testing.T, name string, re *regexp.Regexp, teststr string, expectedresult []string) {
	matchlist := re.FindStringSubmatch(teststr)
	if len(expectedresult) != len(matchlist) {
		t.Errorf("regexp %s did not yield expected results in test: %s but %s", name, teststr, matchlist)
		return
	}
	for c, _ := range expectedresult {
		if expectedresult[c] != matchlist[c] {
			t.Errorf("regexp %s did match expected result in test %s as %s != %s", name, teststr, expectedresult[c], matchlist[c])
		}
	}
}

func TestRegexURLMatch(t *testing.T) {
	mastodon_urls := []string{"https://chaos.social/@qbit/102133941111331502",
		"https://mastodon.social/@test/102133941111331502",
		"https://chaos.social/web/statuses/102140251110038222",
		"http://chaos.social/web/statuses/102140251110038222",
		"https://chaos.social/web/statuses/1"}

	twitter_urls := []string{"https://twitter.com/someone/statuses/1131013299817111553",
		"https://mobile.twitter.com/realraum/status/1131013299817111553",
		"http://mobile.twitter.com/realraum/status/1131013299817111553",
		"https://twitter.com/someone/status/1131013299817111553",
		"http://twitter.com/someone/status/1131013299817111553",
	}

	testMatchRegex(t, "mastodon_status_uri_re_", mastodon_status_uri_re_, mastodon_urls, twitter_urls)
	testMatchRegex(t, "twitter_status_uri_re_", twitter_status_uri_re_, twitter_urls, mastodon_urls)
}

func TestRegexSubmatch(t *testing.T) {
	testSubmatchRegex(t, "mastodon_status_uri_re_", mastodon_status_uri_re_, "https://chaos.social/web/statuses/102140251110038222", []string{"https://chaos.social/web/statuses/102140251110038222", "102140251110038222"})
	testSubmatchRegex(t, "twitter_status_uri_re_", twitter_status_uri_re_, "https://twitter.com/someone/status/1131013299817111553", []string{"https://twitter.com/someone/status/1131013299817111553", "1131013299817111553"})
	testSubmatchRegex(t, "directmsg_re_", directmsg_re_, "blabla @user lala", []string{" @user ", "@user"})
	testSubmatchRegex(t, "directmsg_re_", directmsg_re_, "blabla @user@mastodon.social", []string{" @user@mastodon.social", "@user@mastodon.social"})
	testSubmatchRegex(t, "directmsg_re_", directmsg_re_, "@user@mastodon.social bla bla", []string{"@user@mastodon.social ", "@user@mastodon.social"})
	testSubmatchRegex(t, "directmsg_re_", directmsg_re_, "@user@mastodon.social", []string{"@user@mastodon.social", "@user@mastodon.social"})
}

func TestRegexDM(t *testing.T) {
	testMatchRegex(t, "directmsg_re_", directmsg_re_, []string{"@someone", "@someone@somewhere.social"}, []string{"email@example.com"})
}
