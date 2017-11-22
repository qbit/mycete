# mycete

*VERY ALPHA*

Riot/Matrix room: [#mycete:tapenet.org](https://riot.im/app/#/room/#mycete:tapenet.org)

A [matrix.org](https://matrix.org) micro-blogging (twitter,mastodon) connector.

`mycete` pipes your chat messages from matrix to twitter and/or mastodon. It does this by
listening in on a channel you create. Everything you enter in the channel will be published
to your various feeds!

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
```

## TODO

- [X] TravisCI
- [ ] Read the timelines back into the matrix room.
- [ ] Document the process for getting api keys.
- [ ] Only establish our oauth / auth stuff when a service is enabled.
- [ ] Post to RSS for blogging?
- [ ] Error early if our service is enabled and we have invalid credentials. (See if there is API for testing?)
