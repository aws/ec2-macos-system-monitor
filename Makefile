go=go
goimports=go run golang.org/x/tools/cmd/goimports@latest
golangci_lint=golangci-lint

PKGPATH=$(shell go list -m | head -n1)
T=$(PKGPATH)/...

export GOOS
export GOARCH

.PHONY: all
all: build test

.PHONY: build
build: cpuutilization_darwin

cpuutilization_darwin: cpuutilization_darwin_amd64 cpuutilization_darwin_arm64
	lipo -create -output cpuutilization_darwin cpuutilization_darwin_amd64 cpuutilization_darwin_arm64

cpuutilization_darwin_%:
	GOOS=darwin GOARCH=$* $(go) build $(V) -o cpuutilization_darwin_$* -ldflags="-s -w" .

.PHONY: test
test:
	$(go) test -v -cover $(T)

.PHONY: clean
clean:
	$(go) clean
	rm -f cpuutilization_darwin*

.PHONY: fmt
fmt:: goimports

.PHONY: imports goimports
imports goimports:
	$(goimports) -local $(PKGPATH) $(or $(goimports_flags),-w) .

.PHONY: lint
lint:: golangci-lint

.PHONY: golangci-lint
golangci-lint:
	$(golangci_lint) run
