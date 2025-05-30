module nemu-server

go 1.24.3

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/WJQSERVER-STUDIO/go-utils/copyb v0.0.4
	github.com/WJQSERVER-STUDIO/logger v1.7.2
	github.com/fenthope/gzip v0.0.1
	github.com/fenthope/record v0.0.1
	github.com/infinite-iroha/touka v0.0.4
)

require (
	github.com/WJQSERVER-STUDIO/go-utils/log v0.0.3 // indirect
	github.com/WJQSERVER-STUDIO/httpc v0.5.1 // indirect
	github.com/go-json-experiment/json v0.0.0-20250517221953-25912455fbc8 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
)

replace github.com/fenthope/gzip => /data/github/fenthope/gzip

replace github.com/infinite-iroha/touka => /data/github/WJQSERVER/touka
