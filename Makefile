.PHONY: default lint test

default: test

lint:
	@golint -set_exit_status ./...

test: lint
	@go test -v -race ./...
