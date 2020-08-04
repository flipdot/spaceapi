FROM golang AS builder

COPY . /go/src/github.com/flipdot/spaceapi
WORKDIR /go/src/github.com/flipdot/spaceapi
RUN make
RUN ln -s /dev/stdout spaceapi.log
CMD ["/go/src/github.com/flipdot/spaceapi/spaceapi.fcgi","-local","0.0.0.0:8000"]
