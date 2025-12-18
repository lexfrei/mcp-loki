FROM docker.io/library/golang:1.25-alpine AS builder

ARG VERSION=development

# hadolint ignore=DL3018
RUN echo 'nobody:x:65534:65534:Nobody:/:' > /tmp/passwd && \
    apk add --no-cache upx ca-certificates

WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION}" -trimpath ./cmd/mcp-loki && \
    upx --best --lzma mcp-loki

FROM scratch

COPY --from=builder /tmp/passwd /etc/passwd
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder --chmod=555 /build/mcp-loki /mcp-loki

USER 65534
ENTRYPOINT ["/mcp-loki"]
