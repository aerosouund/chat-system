db:
	@go build -o bin/db db/main.go
	@./bin/db

api:
	@go build -o bin/api ./main.go
	@./bin/api

