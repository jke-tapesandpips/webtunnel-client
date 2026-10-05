FROM golang:1.26-bookworm AS builder

WORKDIR /src

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        git \
        ca-certificates \
        tor \
    && rm -rf /var/lib/apt/lists/* \
    && git clone --depth 1 https://gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/webtunnel.git /src/webtunnel \
    && cd /src/webtunnel/main/client \
    && GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /usr/local/bin/webtunnel . \
    && cd /src \
    && GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /usr/local/bin/webtunnel-check . \
    && mkdir -p /output/lib /output/lib64 /output/usr/lib /output/etc /run/tor \
    && case "${TARGETARCH:-amd64}" in \
        amd64) LIBDIR=x86_64-linux-gnu ;; \
        arm64) LIBDIR=aarch64-linux-gnu ;; \
        *) echo "Unsupported arch: ${TARGETARCH}" >&2; exit 1 ;; \
      esac \
    && cp -a "/lib/${LIBDIR}" /output/lib/ \
    && cp -a /lib/ld-linux* /output/lib/ 2>/dev/null || true \
    && cp -a /lib64 /output/lib64 2>/dev/null || true \
    && cp -a "/usr/lib/${LIBDIR}" /output/usr/lib/ \
    && cp -a /etc/ssl /output/etc/ssl \
    && cp -a /etc/ca-certificates /output/etc/ca-certificates

FROM gcr.io/distroless/base:nonroot

WORKDIR /tmp

COPY --from=builder /usr/local/bin/webtunnel /usr/local/bin/webtunnel
COPY --from=builder /usr/local/bin/webtunnel-check /usr/local/bin/webtunnel-check
COPY --from=builder /usr/bin/tor /usr/bin/tor
COPY --from=builder /etc/tor /etc/tor
COPY --from=builder /usr/share/tor /usr/share/tor
COPY --from=builder --chown=65532:65532 /run/tor /run/tor
COPY --from=builder /output/etc/ssl /etc/ssl
COPY --from=builder /output/etc/ca-certificates /etc/ca-certificates
COPY --from=builder /output/lib /lib
COPY --from=builder /output/lib64 /lib64
COPY --from=builder /output/usr/lib /usr/lib

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/webtunnel-check"]
