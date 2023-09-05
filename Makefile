db:
	@go build -o bin/db db/main.go
	@./bin/db

apiserver:
	@go build -o bin/api api/main.go
	@./bin/api
