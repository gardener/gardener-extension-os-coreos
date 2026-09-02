############# builder
FROM --platform=$BUILDPLATFORM golang:1.27.1 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /go/src/github.com/gardener/gardener-extension-os-coreos
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download
COPY . .
# use build target since cross compiled binaries with `go install` get a $OS_$ARCH directory prefixed, which is problematic for multistage COPY
RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg/mod \
  GOOS=${TARGETOS} GOARCH=${TARGETARCH} make build


############# gardener-extension-os-coreos
FROM gcr.io/distroless/static-debian11:nonroot AS gardener-extension-os-coreos
WORKDIR /

COPY --from=builder /go/src/github.com/gardener/gardener-extension-os-coreos/gardener-extension-os-coreos /gardener-extension-os-coreos
ENTRYPOINT ["/gardener-extension-os-coreos"]
