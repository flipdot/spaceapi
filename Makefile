
BINARY = spaceapi.fcgi
USER = flipdot
SERVER = flipdot.org
DEPLOY_PATH = api.flipdot.org

.PHONY: ${BINARY} deploy deps
${BINARY}:
	go build -ldflags "-s -w" -o $@ spaceapi.go

deps:
	which dep || go get -u github.com/golang/dep/cmd/dep
	dep ensure
	dep prune



