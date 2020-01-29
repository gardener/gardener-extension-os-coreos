############# builder
FROM golang:1.13.4 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-os-coreos
COPY . .
RUN make install-requirements && make VERIFY=true all

############# gardener-extension-os-coreos
FROM builder AS gardener-extension-os-coreos

COPY --from=builder /go/bin/gardener-extension-os-coreos /gardener-extension-os-coreos
ENTRYPOINT ["/gardener-extension-os-coreos"]
