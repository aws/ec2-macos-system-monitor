
PROJECTNAME=$(shell basename "$(PWD)")
VERSION=$(shell git describe --always --tags)

clean:
	go clean
	rm -f *.tar.gz

build:
	@GOOS=darwin GOARCH=amd64 go build -o cpuutilization_darwin -ldflags="-s -w" main.go

tar:
	GZIP=-n tar czf $(PROJECTNAME)-$(VERSION).tar.gz *

test: build
	@GOOS=darwin GOARCH=amd64 go test lib/ec2macossystemmonitor/*.go -v -cover

all: build tar
