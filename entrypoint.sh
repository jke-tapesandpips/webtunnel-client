#!/bin/bash

# Configure Tor with necessary settings
echo "UseBridges 1" > /etc/tor/torrc
echo "SocksPort 1080" >> /etc/tor/torrc
echo "DataDirectory /var/lib/tor" >> /etc/tor/torrc
echo "PIDFile /var/lib/tor/tor.pid" >> /etc/tor/torrc
echo "CookieAuthentication 0" >> /etc/tor/torrc
echo "ClientTransportPlugin webtunnel exec /usr/local/bin/webtunnel" >> /etc/tor/torrc

# Check if CONNECTION_STRING is set and valid
if [ -z "$CONNECTION_STRING" ]; then
    echo "Error: CONNECTION_STRING is not set. Please provide a valid bridge string."
    exit 1
else
    echo "Bridge $CONNECTION_STRING" >> /etc/tor/torrc
fi

# Start Tor with custom configuration
/usr/bin/tor --defaults-torrc /usr/share/tor/tor-service-defaults-torrc --hush

echo $(curl -sS --socks5 localhost:1080 https://check.torproject.org/api/ip)