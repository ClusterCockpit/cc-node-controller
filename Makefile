
SYSFEATURES_FILES = $(wildcard pkg/sysfeatures/*.go)

all: cc-node-controller

cc-node-controller: cmd/server/cc-node-controller.go $(SYSFEATURES_FILES)
	go build -o cc-node-controller cmd/server/cc-node-controller.go

likwid:
	git clone -b v5.3 https://github.com/RRZE-HPC/likwid.git
	cd likwid && make PREFIX=/tmp/likwid-install BUILD_SYSFEATURES=true
	cd likwid && sudo make PREFIX=/tmp/likwid-install BUILD_SYSFEATURES=true install
	echo "export LD_LIBRARY_PATH=/tmp/likwid-install/lib:$$LD_LIBRARY_PATH"

