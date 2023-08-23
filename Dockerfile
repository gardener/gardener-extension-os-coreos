############# builder
FROM golang:1.21.0 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-os-coreos
COPY . .
RUN make install

############# gardener-extension-os-coreos
FROM gcr.io/distroless/static-debian11:nonroot AS gardener-extension-os-coreos
WORKDIR /

COPY --from=builder /go/bin/gardener-extension-os-coreos /gardener-extension-os-coreos
ENTRYPOINT ["/gardener-extension-os-coreos"]
