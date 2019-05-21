# mycete

<a href="https://github.com/qbit/mycete"><img src="https://raw.githubusercontent.com/qbit/mycete/master/logo.png" align="left" height="48" width="48" ></a>

[![Build Status](https://travis-ci.org/qbit/mycete.svg?branch=master)](https://travis-ci.org/qbit/mycete)

*BETA: Software is used frequently but Bugs may still appear and behaviour may change at any commit*

Riot/Matrix room: [#mycete:tapenet.org](https://riot.im/app/#/room/#mycete:tapenet.org)

A [matrix.org](https://matrix.org) micro-blogging (twitter,mastodon) connector.

`mycete` pipes your chat messages from matrix to twitter and/or mastodon. It does this by
listening in on a channel you create. Everything you enter in the channel will be published
to your various feeds!

Optionaly, only stuff you prepend with a ''guard_prefix'' will be published. Obviously the prefix will be removed first.

Delete tweets and toots you posted by redacting the corresponding matrix message.

If you upload images to the controlling matrix room, they will be appended to your next toot and tweet.

Tweets and Toots may be favoured or reblogged / retweeted by using the `reblog_cmd` or `favourite_cmd` (specified in the `[matrix]` section) followed by the status URL or ID

## Example Information Flow

<img src="https://raw.githubusercontent.com/btittelbach/lightningtalks_mycete-mastodonboostbot-matrix/master/images/mycete_statusflow.png" align="center" style="width:100%;">

see also slides of lightningtalks which can be found [here](https://github.com/btittelbach/lightningtalks_mycete-mastodonboostbot-matrix)


## from Mastodon back to Matrix

`mycete` will also (optinally) inform you about toots that did not originate from `mycete` as well as when someone favourites or reblogs your status and when someone follows you.

The controlling settings are `show_mastodon_notifications`, `show_own_toots_from_foreign_clients` and 
`show_complete_home_stream` in `[matrix]`

If you don't need this, just remove the `feed2matrix` section.

Additionally it is possible to mirror your complete homestream or just part of it to other matrix rooms.
For each room you may filter by tag, post visibility, sensitivity, weather it is an original toot or a reblog, weather our account posted it or someone else and weather or not we are following the author.

To do this, create a separate configuration section for each room named `feed2morerooms_xxxxx` where xxxxx is your name for that configuration. You can specify arbitrary many configurations, as only the ones listed in `[feed2morerooms]configurations` are activated and used.

In addition to the home stream, it is possible to subscribe tag streams using `[feed2morerooms]subscribe_tagstreams` which will be mixed together with the homestream into one big stream which your configurations (s.a.) will then filter.

If you don't need this, just leave `configurations` empty or remove all `feed2morerooms` sections.

## Building

```
git clone https://github.com/qbit/mycete
cd mycete
go build
```

## Example Config

```
[server]
twitter=true
mastodon=true

[matrix]
user=@fakeuser:matrix.org
password=snakesonaplane
url=https://matrix.org
room_id=!iasdfadsfadsfafs:matrix.org
guard_prefix=t>
reblog_prefix=reblog>
favourite_prefix=+1>
directtoot_prefix=dm>
directtootreply_prefix=reply>
directtweet_prefix=tdm>
help_prefix=!help
join_welcome_text="Welcome! Warning: Everything you say I will toot and/or tweet to the world if it starts with t>"
admins_can_redact_user_status=false

[twitter]
consumer_key=
consumer_secret=
access_token=
access_secret=

[mastodon]
server=https://mastodon.social
client_id=
client_secret=
access_token=

[images]
enabled=true
temp_dir=/tmp

[feed2matrix]
show_mastodon_notifications=true
show_own_toots_from_foreign_clients=true
show_complete_home_stream=false
characterlimit = 1000
imagebyteslimit = 4194304
imagecountlimit = 4

[feed2morerooms]
subscribe_tagstreams=interesstingtag otherinteresstingtag
configurations=filter1 filter2

[feed2morerooms_filter1]
target_room=!example1:matrix.org
filter_visibility=public
filter_for_tags=interesstingtag
filter_sensitive=false
filter_reblogs=false
filter_myposts=true
filter_otherpeoplesposts=false
filter_unfollowed=false

[feed2morerooms_filter2]
target_room=!example2:matrix.org
filter_visibility=public
filter_for_tags=otherinteresstingtag
filter_sensitive=false
filter_reblogs=true
filter_myposts=false
filter_otherpeoplesposts=false
filter_unfollowed=true


```

## Linking to Mastodon

When logged into your Mastodon Account in your web browser, go to "Settings", then "Development", then "Your Applications". Create a New Application and give it the required permissions. Put `Client key`, `Client secret` and `Your access token` the tokens into your 'mycete' configuration.

### required permissions
read:accounts read:blocks read:favourites read:filters read:follows read:lists read:mutes read:notifications read:search read:statuses write:conversations write:favourites write:filters write:media write:statuses push

## Linking to Twitter

Oauth via console pin. (TODO)

## TODO

- [ ] create an interface for clients.
- [X] TravisCI.
- [X] Read the timelines back into the matrix room.
- [X] favorite and reblog Mastodon status
- [X] un-reblog and un-favourite when redacting matrix message
- [ ] tests
- [ ] Document the process for getting api keys.
- [ ] Only establish our oauth / auth stuff when a service is enabled.
- [ ] Post to RSS for blogging? (Mastodon already does rss out of the box)
- [ ] Error early if our service is enabled and we have invalid credentials. (See if there is API for testing?)
- [X] post images
- [X] support uploading multiple images per Toot/Tweet
- [X] more feedback and user error guards
- [X] use constrained memory, not slowly ever growing maps. Aka don't be a memory hog
- [ ] twitter stream to matrix, favorite and retweet
- [ ] look into support for small videos
- [ ] clean up matrixbot.go prefix parser code
- [ ] find a way to boost/replyto/favourite remote Toots (requires translation of URL to local Mastodon instance's status ID). In the meantime we add a "reply using this" URL in the room
- [ ] make showing images in Matrix rooms optional for each additional room
