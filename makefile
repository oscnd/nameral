include .env

.PHONY: protoc
protoc:
	protoc --go_out=./generate --go-grpc_out=./generate ./_proto/*.proto

.PHONY: clean
clean:
	git clean -fdX

.PHONY: build
build:
	env GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o .local/backend .

.PHONY: build-windows
build-windows:
	env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o .local/backend.exe .