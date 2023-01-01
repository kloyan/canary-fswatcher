BIN_DIR?=dist/

default: build

.PHONY: clean
clean:
	rm -rfv dist/ *.o

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint: fmt
	golint ./...

.PHONY: vet
vet: fmt
	go vet ./...

.PHONY: build
build: vet
	CGO_ENABLED=0 go build -o $(BIN_DIR) .
