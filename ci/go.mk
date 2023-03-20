GO = go
GOIMPORTS = goimports $(if $(GOPKGPATH),-local $(GOPKGPATH))
GOLANGCILINT = golangci-lint

export GOOS
export GOARCH
export CGO_ENABLED

goget:
	$(GO) get -t ./...

gobuild:
	$(GO) build $(GO_BUILD_FLAGS) $(V) ./...

gotest:
	$(GO) test $(GO_TEST_FLAGS) $(V) ./...

goimports gofmt:
	$(GOIMPORTS) -w .

golint:
	$(GOLANGCILINT) run
