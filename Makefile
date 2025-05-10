.PHONY: run
run:
	cd cmd && ./main

.PHONY: build
build:
	cd cmd && go build -o main

.PHONY: start
start: build run

.PHONY: test
test:
	go test -count=1 -v ./...

.PHONY: generate
generate:
	protoc --go_out=. --go_opt=paths=source_relative \
           --go-grpc_out=. --go-grpc_opt=paths=source_relative \
           internal/proto/pubsub.proto