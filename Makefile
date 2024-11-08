BRANCH ?= main
BUILD_N ?= 0

build:
	@rm -rf bin/*
	go build -ldflags="-X 'main.Version=1.0.0.$(BUILD_N)-$(BRANCH)'" -o ./bin/dev-proxy ./cmd/dev-proxy

run: 
	go run cmd/dev-proxy/main.go

test: 
	go test ./...