# stargazers

## Features

- monitor the star events of the GitHub repo
- send the notifications to Slack or Lark

## How to use

For Lark, create a bot called like `stargazers`, and use this bot to send notifications to you.

For Slack, create an app called like `stargazers`, and add this app into an channel.

Run `stargazers`:

`stargazers -f config.yaml`

`config.yaml` looks like:

```yaml
token: <github token>
repo: <github repo like zeromicro/go-zero>
interval: 1m
trending:
  language: Go
  dateRanges:
    - daily
    - weekly
    - monthly
lark:
  appId: <app id>
  appSecret: <app secret>
  receiver: <receiver's lark UserID>
  receiver_email: <receiver's lark Email>
slack:
  token: <oauth token>
  channel: <channel>
```

The notification message looks like:

- star event
```
stars: 12157
today: 27
user: <user>
name: <name>
followers: 6
time: 10-26 22:52:56
```

- trending event
```
go-zero
Go daily trending: 9
Go weekly trending: 5
Go monthly trending: 19
```
