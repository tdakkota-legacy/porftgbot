# porftgbot
Simple Telegram inline query bot using https://porfirevich.ru/.


## Installation 
Install bot using `go get`

```bash
go get github.com/tdakkota/porftgbot/cmd/porftgbot
```

[Create Telegram application and get `api_id` and `api_hash`](https://core.telegram.org/api/obtaining_api_id).

Set `APP_ID`, `APP_HASH` ,`BOT_TOKEN` environment variables and run(it needs to be `$GOPATH/bin` in your `$PATH`)
```bash
porftgbot run
```
