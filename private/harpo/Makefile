BINARY := harpo
WIN_BINARY := harpo.exe
PREFIX ?= $(HOME)/.local/bin

.PHONY: build build-windows build-windows-arm64 static test install uninstall clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BINARY) ./cmd/harpo

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(WIN_BINARY) ./cmd/harpo

build-windows-arm64:
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o harpo-arm64.exe ./cmd/harpo

static: build

test:
	go test -v ./...

install: build
	install -d $(PREFIX)
	install -m 0755 $(BINARY) $(PREFIX)/$(BINARY)
	ln -sf $(BINARY) $(PREFIX)/harpo-run
	@echo "Installed $(PREFIX)/$(BINARY) and symlink $(PREFIX)/harpo-run"

uninstall:
	rm -f $(PREFIX)/$(BINARY) $(PREFIX)/harpo-run

clean:
	rm -f $(BINARY) $(WIN_BINARY) harpo-arm64.exe
