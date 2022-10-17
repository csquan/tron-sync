all: build

CURRENT_DIR=$(pwd)

PROJ = heco-sync
MODULE = "heco-sync"

PKG = `go list ./... | grep -v /vendor/`

#print tag into pkg
PKG_TAG = -ldflags "-w -X ${PROJ}/common.gitTag=`git describe --tags` -X ${PROJ}/common.commitNumber=`git rev-parse HEAD` -X ${PROJ}/common.buildTime=`date +%FT%T%z`"

#cross compile
CROSS_COMPILE = CGO_ENABLED=0 GOOS=linux GOARCH=amd64
CILINT := $(shell command -v golangci-lint 2> /dev/null)
GOIMPORTS := $(shell command -v goimports 2> /dev/null)

style:
	! find . -path ./vendor -prune -o -name '*.go' -print | xargs goimports -d -local ${MODULE} | grep '^'

format:
ifndef GOIMPORTS
	$(error "goimports is not available please install goimports")
endif
	find . -path ./vendor -prune -o -name '*.go' -print | xargs goimports -l -local ${MODULE} | xargs goimports -l -local ${MODULE} -w

cilint:
	golangci-lint run

clean:
	rm -rf bin

build:
	go build ${PKG_TAG} -o bin/heco-sync main.go

test: style cilint
	go test -cover ./...

server: clean test
	${CROSS_COMPILE} go build -o bin/linux-amd64-heco-sync ${PKG_TAG} main.go

comp_balance_server:
	${CROSS_COMPILE} go build -o bin/comp_balance ${PKG_TAG} tools/compensation/*.go

.PHONY: build clean client