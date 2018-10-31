# https://azer.bike/journal/a-good-makefile-for-go/

# Go related variables.
GOFILES=$(wildcard *.go)
PROJECTNAME=$(shell basename "$(PWD)")

## compile: Compile the binary.
compile: go-compile

## clean: Clean build files. Runs `go clean` internally.
clean: go-clean

go-compile: go-clean dep-ensure go-test go-build

go-build:
	@echo "  >  Building binary..."
	go build -o bin/$(PROJECTNAME) $(GOFILES)

dep-ensure:
	@echo "  >  Checking if there is any missing dependencies..."
	dep ensure -v -vendor-only

go-clean:
	@echo "  >  Cleaning build cache"
	go clean

go-test:
	@echo "  >  Running tests"
	go test -v

.PHONY: help
all: help
help: Makefile
	@echo
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
