module github.com/fslongjin/liteboxd/sdk/go

go 1.24.0

require (
	github.com/fslongjin/liteboxd/backend v0.0.0-20260303131241-3779310d3fc3
	github.com/gorilla/websocket v1.5.3
)

replace github.com/fslongjin/liteboxd/backend => ../../backend
