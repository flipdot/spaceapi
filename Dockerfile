FROM golang:1.24 AS builder

COPY . /app
WORKDIR /app
RUN make
RUN ln -s /dev/stdout spaceapi.log
CMD ["/app/spaceapi.fcgi","-local","0.0.0.0:8000"]
