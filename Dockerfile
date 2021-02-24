############# builder
FROM golang:1.15.8 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-os-coreos
COPY . .
RUN make install

############# gardener-extension-os-coreos
FROM alpine:3.13.2 AS gardener-extension-os-coreos

COPY --from=builder /go/bin/gardener-extension-os-coreos /gardener-extension-os-coreos
ENTRYPOINT ["/gardener-extension-os-coreos"]
