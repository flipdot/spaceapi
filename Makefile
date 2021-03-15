BINARY = spaceapi.fcgi
USER = flipdot
SERVER = flipdot.org
DEPLOY_PATH = api.flipdot.org

PACKAGE = github.com/flipdot/spaceapi

.PHONY: ${BINARY} deploy deps
${BINARY}:
	go build -ldflags "-s -w" -o $@ ${PACKAGE}
	chmod 0755 $@ # this is crucial for fcgi to work



