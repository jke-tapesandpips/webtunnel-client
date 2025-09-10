FROM golang:latest

RUN apt-get update && apt-get install -y \
    git \
    tor \
    curl \
    minio-client \
    jq \
    && rm -rf /var/lib/apt/lists/*

RUN ln -s /usr/bin/minio-client /usr/local/bin/mc

RUN git clone https://gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/webtunnel \
    && cd webtunnel/main/client \
    && go build \
    && cp client /usr/local/bin/webtunnel

COPY test-webtunnel-connections.sh /usr/local/bin/test-webtunnel-connections.sh
RUN chmod +x /usr/local/bin/test-webtunnel-connections.sh

WORKDIR /app