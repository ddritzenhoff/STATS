# statsd

```json
{
"db":{
    "dsn": ""
},
"http":{
    "addr":""
},
"slack":{
    "signing_secret": "",
    "bot_signing_key": "",
    "channel_id": ""
}
}
```

`dsn`: The data source name (dsn) represents the path to the sqlite database.
`addr`: The http server will listen on this address.
`signing_secret`: Used to verify Slack requests.
`bot_signing_key`: Used to send messages into the Slack workspace.
`channel_id`: Used to send messages into a specific channel.
