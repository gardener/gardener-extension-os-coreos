#############      builder       #############
FROM golang:1.11.4 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-os-coreos
COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go install \
  -ldflags "-X github.com/gardener/gardener-extension-os-coreos/pkg/version.Version=$(cat VERSION)" \
  ./...

#############      extension     #############
FROM alpine:3.8 AS extension

RUN apk add --update bash curl

COPY --from=builder /go/bin/gardener-extension-os-coreos /gardener-extension-os-coreos

WORKDIR /

ENTRYPOINT ["/gardener-extension-os-coreos"]
