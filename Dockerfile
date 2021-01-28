############# builder
FROM eu.gcr.io/gardener-project/3rd/golang:1.15.7 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-os-coreos
COPY . .
RUN make install

############# gardener-extension-os-coreos
FROM eu.gcr.io/gardener-project/3rd/alpine:3.12.3 AS gardener-extension-os-coreos

COPY --from=builder /go/bin/gardener-extension-os-coreos /gardener-extension-os-coreos
ENTRYPOINT ["/gardener-extension-os-coreos"]
