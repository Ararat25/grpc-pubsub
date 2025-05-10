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