VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  = -ldflags "-s -w -X github.com/xunull/nght/cmd.Version=$(VERSION)"

.PHONY: build test vet fmt fmt-check clean release release-snapshot

build:
	go build $(LDFLAGS) -o nght .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@out=$$(gofmt -l . 2>&1); if [ -n "$$out" ]; then echo "gofmt issues:"; echo "$$out"; exit 1; fi

clean:
	rm -rf nght dist/

release:
	goreleaser release --clean

release-snapshot:
	goreleaser release --snapshot --clean
