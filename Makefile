.PHONY: swagger run

swagger:
	swag init -g pkg/swagger/swagger.go -d ./ -o docs --parseInternal

run: swagger
	go run ./cmd/app
