#!/usr/bin/env bash

if [ -z "$1" ]; then
    echo "Usage: $0 inputfile.txt"
    exit 1
fi

input_file="$1"
base_dir="/var/lib/tor"

line_number=1
while IFS= read -r LINE_CONTENT || [[ -n "$LINE_CONTENT" ]]; do
    config_file="/etc/tor/torrc${line_number}"
    data_dir="${base_dir}${line_number}"
    socks_port=$((1080 + line_number - 1))
    pid_file="${data_dir}/tor.pid"
    comment=$(echo "$LINE_CONTENT" | sed -n 's/.*#//p')

    # Ensure unique data directory
    mkdir -p "$data_dir"
    chown -R debian-tor:debian-tor "$data_dir"

    # Write config
    {
        echo "UseBridges 1"
        echo "SocksPort ${socks_port}"
        echo "DataDirectory ${data_dir}"
        echo "PIDFile ${pid_file}"
        echo "CookieAuthentication 0"
        echo "ClientTransportPlugin webtunnel exec /usr/local/bin/webtunnel"
        echo "Bridge ${LINE_CONTENT}"
    } > "$config_file"

    # Start tor instance in background
    /usr/bin/tor \
        --defaults-torrc /usr/share/tor/tor-service-defaults-torrc \
        -f "$config_file" --hush >/dev/null 2>&1 &

    # Give it some time to bootstrap
    sleep 10

    # Test Tor connection
    is_tor=$(curl -sS --socks5 localhost:1080 https://check.torproject.org/api/ip | jq -r '.IsTor')

    # Print result with comment
    if [ "$is_tor" = "true" ]; then
        # Green
        echo -e "✅ \033[0;32mTor connection is $is_tor for $comment\033[0m"
    else
        # Red
        echo -e "❌ \033[0;31mTor connection is $is_tor for $comment\033[0m"
    fi

    ((line_number++))
done < "$input_file"