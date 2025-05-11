# Определение оболочки запуска
UNAME := $(shell uname 2>/dev/null)

# Определение платформы
ifeq ($(OS),Windows_NT)
    ifneq (,$(findstring MINGW,$(UNAME)))
        BINARY := main
    else
        BINARY := main.exe
    endif
else
    BINARY := main
endif

.PHONY: build
build:
	cd cmd && go build -o $(BINARY)

.PHONY: test
test:
	go test -count=1 -v ./...

.PHONY: generate
generate:
	protoc --go_out=. --go_opt=paths=source_relative \
           --go-grpc_out=. --go-grpc_opt=paths=source_relative \
           internal/proto/pubsub.proto