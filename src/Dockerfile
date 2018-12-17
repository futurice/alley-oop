FROM golang:1.10.0-alpine AS builder
LABEL maintainer="fiba@futurice.com"

# Install from git to satisfy dependencies etc:
RUN apk add --update git
RUN go get github.com/futurice/alley-oop/src

# Overwrite with possible local changes (makes dev less painful):
ADD . /go/src/github.com/futurice/alley-oop/

# Build the binary:
WORKDIR /go/src/github.com/futurice/alley-oop/src
RUN go build

# Start from a clean slate for the actual image:
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /go/src/github.com/futurice/alley-oop/src/src alley-oop
RUN apk --no-cache add ca-certificates && update-ca-certificates

VOLUME ["/etc/alley-oop", "/var/lib/alley-oop"]
ENTRYPOINT ["./alley-oop", "/etc/alley-oop/config.cfg"]
EXPOSE 53 53/udp 80 443
