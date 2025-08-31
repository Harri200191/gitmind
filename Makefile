BINARY_NAME=commitgen
DIST_DIR=dist

.PHONY: build install clean

build:
	GO111MODULE=on CGO_ENABLED=1 go build -o $(DIST_DIR)/$(BINARY_NAME) ./cmd/commitgen

install: build
	install -m 0755 $(DIST_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

clean:
	rm -rf $(DIST_DIR)