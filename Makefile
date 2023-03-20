GOPKGPATH = github.com/aws/ec2-macos-system-monitor

.PHONY: all
all: build test

.PHONY: build
# we use go-psutil which builds against headers
build: CGO_ENABLED=1
build: cpuutilization_darwin

.PHONY: test
test: GO_TEST_FLAGS=-cover
test: gotest

.PHONY: lint
lint: golint

.PHONY: format fmt
format fmt: goimports

cpuutilization_darwin: cpuutilization_darwin_amd64 cpuutilization_darwin_arm64
	lipo -create -output $@ $^

cpuutilization_darwin_%:
	GOOS=darwin GOARCH=$* \
		go build -o cpuutilization_darwin_$* -ldflags="-s -w" .

.PHONY: clean
clean:
	go clean
	-rm -f cpuutilization_darwin*

include ci/go.mk
