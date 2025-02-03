
SYSFEATURES_FILES = $(wildcard pkg/sysfeatures/*.go)
SERVER_FILES = $(wildcard cmd/server/*.go)

all: cc-node-controller

cc-node-controller: $(SERVER_FILES) $(SYSFEATURES_FILES)
	go build -o cc-node-controller ./cmd/server/

likwid:
	git clone -b v5.4.1 https://github.com/RRZE-HPC/likwid.git
	cd likwid && make PREFIX=/tmp/likwid-install BUILD_SYSFEATURES=true
	cd likwid && sudo make PREFIX=/tmp/likwid-install BUILD_SYSFEATURES=true install
	echo "export LD_LIBRARY_PATH=/tmp/likwid-install/lib:$$LD_LIBRARY_PATH"

.PHONY: clean
clean:
	rm --force cc-node-controller
