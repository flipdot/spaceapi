
BINARY = spaceapi.fcgi
USER = flipdot
SERVER = flipdot.org
DEPLOY_PATH = api.flipdot.org
GOPATH = $(shell pwd)/.gopath
export GOPATH
FAKE_GOPATH = ${GOPATH}/src/github.com/flipdot/

PACKAGE = github.com/flipdot/spaceapi

.PHONY: ${BINARY} deploy deps
${BINARY}: ${FAKE_GOPATH}
	go get -v ${PACKAGE}
	go build -ldflags "-s -w" -o $@ ${PACKAGE}

${FAKE_GOPATH}:
	mkdir -p $@
	ln -s ../../../../ ${GOPATH}/src/${PACKAGE}

deps:
	which dep || go get -u github.com/golang/dep/cmd/dep
	dep ensure
	dep prune



