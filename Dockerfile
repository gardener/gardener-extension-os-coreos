############# builder
FROM golang:1.14.4 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-os-coreos
COPY . .
RUN make install

############# gardener-extension-os-coreos
FROM alpine:3.12.0 AS gardener-extension-os-coreos

COPY --from=builder /go/bin/gardener-extension-os-coreos /gardener-extension-os-coreos
ENTRYPOINT ["/gardener-extension-os-coreos"]
