
SYSFEATURES_FILES = $(wildcard pkg/sysfeatures/*.go)
SERVER_FILES = $(wildcard cmd/server/*.go)

all: cc-node-controller

cc-node-controller: $(SERVER_FILES) $(SYSFEATURES_FILES)
	go build -o cc-node-controller ./cmd/server/

.PHONY: clean DEB RPM
clean:
	rm --force cc-node-controller

DEB:
	./scripts/makedeb.sh

RPM:
	./scripts/makerpm.sh
