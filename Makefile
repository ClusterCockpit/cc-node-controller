.PHONY: all cc-node-controller clean DEB RPM
all: cc-node-controller

cc-node-controller:
	go build -o cc-node-controller ./cmd/server/

clean:
	rm --force cc-node-controller

DEB:
	./scripts/makedeb.sh

RPM:
	./scripts/makerpm.sh
