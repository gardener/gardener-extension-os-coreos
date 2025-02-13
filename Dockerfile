############# builder
FROM golang:1.24.0 AS builder

WORKDIR /go/src/github.com/gardener/gardener-extension-os-coreos
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make install

############# gardener-extension-os-coreos
FROM gcr.io/distroless/static-debian11:nonroot AS gardener-extension-os-coreos
WORKDIR /

COPY --from=builder /go/bin/gardener-extension-os-coreos /gardener-extension-os-coreos
ENTRYPOINT ["/gardener-extension-os-coreos"]
