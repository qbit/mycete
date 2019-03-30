# mycete

<a href="https://github.com/qbit/mycete"><img src="https://raw.githubusercontent.com/qbit/mycete/master/logo.png" align="left" height="48" width="48" ></a>

[![Build Status](https://travis-ci.org/qbit/mycete.svg?branch=master)](https://travis-ci.org/qbit/mycete)

*STILL ALPHA*

Riot/Matrix room: [#mycete:tapenet.org](https://riot.im/app/#/room/#mycete:tapenet.org)

A [matrix.org](https://matrix.org) micro-blogging (twitter,mastodon) connector.

`mycete` pipes your chat messages from matrix to twitter and/or mastodon. It does this by
listening in on a channel you create. Everything you enter in the channel will be published
to your various feeds!

Optionaly, only stuff you prepend with a ''guard_prefix'' will be published. Obviously the prefix will be removed first.

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
join_welcome_text="Welcome! Warning: Everything you say I will toot and/or tweet to the world if it starts with t>"
admins_can_redact_user_status=false
show_mastodon_notifications=true
show_own_toots_from_foreign_clients=true
show_complete_home_stream=false

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
dst_room_ids=
subscribe_only_public=false
characterlimit = 89
filter_tags=
filter_sensitive=false

[feed2matrix]
characterlimit = 1000
target_room_ids=
filter_visibility=public unlisted private direct
filter_for_tags=
filter_sensitive=false
filter_reblogs=false
filter_otherpeoplesposts=true
subscribe_tagstreams=

```

## TODO

- [ ] create an interface for clients.
- [X] TravisCI.
- [X] Read the timelines back into the matrix room.
- [ ] Document the process for getting api keys.
- [ ] Only establish our oauth / auth stuff when a service is enabled.
- [ ] Post to RSS for blogging?
- [ ] Error early if our service is enabled and we have invalid credentials. (See if there is API for testing?)
- [X] post images
- [X] more feedback and user error guards
