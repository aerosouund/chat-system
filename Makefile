consumer:
	@go build -o bin/consumer consumer/main.go
	@./bin/consumer

apiserver:
	@go build -o bin/api api/main.go
	@./bin/api