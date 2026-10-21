VERSION ?= $(shell git tag -l | grep -E '^v?[0-9]+\.[0-9]+\.[0-9]+$$' | sort -V | tail -n 1 || echo "v1.3.0")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
DATE ?= $(shell date -u +%Y-%m-%d)
LDFLAGS = -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: build test changelog shrink install clean uninstall

build:
	go build $(LDFLAGS) -o leakjs ./cmd/leakjs

test:
	go test -v ./...

changelog:
	./scripts/generate-changelog.sh

shrink:
	./scripts/shrink-repo.sh

install: build
	sudo mv leakjs /usr/local/bin/
	@echo "LeakJS installed successfully"

clean:
	rm -f leakjs
	rm -rf dist/

uninstall:
	sudo rm -f /usr/local/bin/leakjs
	@echo "LeakJS uninstalled"
