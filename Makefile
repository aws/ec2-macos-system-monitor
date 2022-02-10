.PHONY: all
all: build test

.PHONY: build
build: cpuutilization_darwin

cpuutilization_darwin: cpuutilization_darwin_amd64 cpuutilization_darwin_arm64
	lipo -create -output cpuutilization_darwin cpuutilization_darwin_amd64 cpuutilization_darwin_arm64

cpuutilization_darwin_%:
	GOOS=darwin GOARCH=$* go build -o cpuutilization_darwin_$* -ldflags="-s -w" .

.PHONY: test
test:
	GOOS=darwin go test -v -cover ./...

.PHONY: clean
clean:
	go clean
	rm -f cpuutilization_darwin*
