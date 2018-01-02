#

.PHONY: prepare relay clean vendor relay-darwin

GOCMD=go
GOBUILD=$(GOCMD) build -ldflags -s -v
BINARY_NAME=relay

prepare:
	/bin/sh build/prepare.sh

relay:prepare
	$(GOBUILD) -o build/bin/$(BINARY_NAME) cmd/lrc/*
	@echo "It's done. You can run build/bin/$(BINARY_NAME) now."

clean:
	rm build/bin/*

vendor:
	/bin/bash vendor.sh

relay-darwin:prepare
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 $(GOBUILD) -o build/bin/$(BINARY_NAME)_darwin cmd/lrc/*
	@echo "done"

relay-linux-amd64:prepare
	GOOS=linux CGO_ENABLED=1 GOARCH=amd64 $(GOBUILD) -o build/bin/$(BINARY_NAME)_linux_amd64 cmd/lrc/*
	@echo "done"

relay-linux-386:prepare
	GOOS=linux GOARCH=386 CGO_ENABLED=1 $(GOBUILD) -o build/bin/$(BINARY_NAME)_linux_386 cmd/lrc/*
	@echo "done"

relay-windows-amd64:prepare
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 $(GOBUILD) -o build/bin/$(BINARY_NAME)_windows_amd64 cmd/lrc/*
	@echo "done"

relay-windows-386:prepare
	GOOS=windows GOARCH=386 CGO_ENABLED=1 $(GOBUILD) -o build/bin/$(BINARY_NAME)_windows_386 cmd/lrc/*
	@echo "done"

