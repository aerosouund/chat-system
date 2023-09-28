consumer:
	@go build -o bin/cons consumer/main.go
	@./bin/cons

apiserver:
	@go build -o bin/api api/*
	@./bin/api

.PHONY: apiserver consumer