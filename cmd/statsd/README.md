# statsd

```json
{
"listen_address": "localhost:8080",
"dsn": "/home/yourusername/databases/stats.db",
"slack":{
    "signing_secret": "",
    "bot_signing_key": "",
    "channel_id": ""
}
}
```

`listen_address`: listen address
`dsn`: The data source name (dsn) represents the path to the sqlite database.
`signing_secret`: Used to verify Slack requests
`bot_signing_key`: Used to send messages into the Slack workspace
`channel_id`: Used to send messages into a specific channel
