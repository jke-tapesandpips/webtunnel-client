FROM golang:latest

ENV CONNECTION_STRING=""

RUN apt-get update && apt-get install -y \
    git \
    tor \
    curl \
    && rm -rf /var/lib/apt/lists/*

RUN git clone https://gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/webtunnel \
    && cd webtunnel/main/client \
    && go build \
    && cp client /usr/local/bin/webtunnel

# Copy the script to the container
COPY entrypoint.sh /usr/local/bin/entrypoint.sh

# Make the script executable
RUN chmod +x /usr/local/bin/entrypoint.sh

# Set the script as the entrypoint
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

CMD ["/bin/bash"]