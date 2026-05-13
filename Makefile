BINARY  := authentik-exporter
PKG     := ./cmd/authentik-exporter
IMAGE   := authentik-exporter
TAG     ?= dev

GO_LDFLAGS := -s -w
LDFLAGS    := -trimpath -ldflags='$(GO_LDFLAGS)'

# Platforms built by `make build-all`. Override on the command line, e.g.
#   make build-all PLATFORMS="linux/amd64 linux/arm64"
PLATFORMS ?= linux/amd64 linux/arm64 darwin/arm64

export CGO_ENABLED := 0

.PHONY: build build-all test vet lint run docker clean

build:
	go build $(LDFLAGS) -o bin/$(BINARY) $(PKG)

# Pattern target: `make build-linux-amd64` → bin/authentik-exporter-linux-amd64
# `$*` is everything after `build-`, e.g. "linux-amd64".
build-%:
	@os=$$(echo $* | cut -d- -f1); arch=$$(echo $* | cut -d- -f2); \
	echo ">> building $$os/$$arch"; \
	GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o bin/$(BINARY)-$$os-$$arch $(PKG)

# Build every entry in PLATFORMS.
build-all: $(foreach p,$(PLATFORMS),build-$(subst /,-,$(p)))

test:
	go test ./...

vet:
	go vet ./...

lint: vet
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; else echo "golangci-lint not installed, skipping"; fi

run: build
	./bin/$(BINARY)

docker:
	docker buildx build --load -t $(IMAGE):$(TAG) .

clean:
	rm -rf bin
