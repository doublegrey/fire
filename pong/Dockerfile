FROM golang:alpine AS builder
RUN apk update && apk add --no-cache git
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /build
COPY . .
RUN go build -o /go/bin/pong
FROM scratch
COPY --from=builder  /go/bin/pong /go/bin/pong
ENTRYPOINT ["/go/bin/pong"]
