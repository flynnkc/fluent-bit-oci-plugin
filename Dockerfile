# Build stage
FROM golang:1.23.3-alpine AS builder
RUN apk add --no-cache make cmake gcc g++ libc-dev
WORKDIR /src
COPY . .
RUN go build -buildmode=c-shared -o fb_oci.so *.go

# Run stage
FROM fluent/fluent-bit:latest
WORKDIR /fluent-bit
COPY --from=builder /src/fb_oci.so /fluent-bit/bin/fb_oci.so
ENV FLB_PLUGIN_PATH="/fluent-bit/bin/fb_oci.so"
ENTRYPOINT ["/fluent-bit/bin/fluent-bit", "-e", "/fluent-bit/bin/fb_oci.so", "-o", "fb_oci_logging", "-i", "dummy"]