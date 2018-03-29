FROM golang:1.10.0-alpine AS builder
LABEL maintainer="fiba@futurice.com"

RUN apk add --update gcc musl-dev git
RUN go get github.com/futurice/alley-oop
WORKDIR /go/src/github.com/futurice/alley-oop
RUN CGO_ENABLED=1 go build

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /go/src/github.com/futurice/alley-oop .

ENTRYPOINT ["./alley-oop"]
EXPOSE 53 53/udp 80 443
