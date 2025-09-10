# Tor Webtunnel Client
A docker container to test your webtunnel "bridges"

## Manual Testing

Usage:
```
cd manual
docker build . -t <your-tag>
docker run --rm -it -e CONNECTION_STRING="webtunnel 10.0.0.161:443 50M3F1N93RPR1N7 url=https://webtunnel.example.com/your-webtunnel-path ver=0.0.1" <your-tag>
```
Or you can use the prebuilt image:
```
docker run --rm -it -e CONNECTION_STRING="webtunnel 10.0.0.161:443 50M3F1N93RPR1N7 url=https://webtunnel.example.com/your-webtunnel-path ver=0.0.1" bubatzlegal/webtunnel-client:manual-latest
```

Example Output:
```
{"IsTor":true,"IP":"45.84.107.47"}
```
This is the default output you would get from [https://check.torproject.org/api/ip](https://check.torproject.org/api/ip)

## Automated Testing
Usage:

### IMPORTANT
The input file needs to look like [the example input file](automated/example-input-file.txt). The comment in the end of the line is parsed and used for the output.

```
cd automated
docker build . -t <your-tag>
docker run --rm -it -v ${PWD}/automated/example-input-file.txt:/app/example-input-file.txt <your-tag> test-webtunnel-connections.sh /app/example-input-file.txt
```
Or you can use the prebuilt image:
```
docker run --rm -it -v ./automated/example-input-file.txt:/app/example-input-file.txt bubatzlegal/webtunnel-client:automated-latest test-webtunnel-connections.sh /app/example-input-file.txt
docker run --rm -it -v ./connection-strings.txt:/app/connection-strings.txt bubatzlegal/webtunnel-client:automated-latest test-webtunnel-connections.sh /app/connection-strings.txt
```

Example Output:
```
✅ Tor connection is true for bridge-1 # Check Tor API-Response: {"IsTor":true,"IP":"..."}
❌ Tor connection is false for bridge-2 # Check Tor API-Response: {"IsTor":false,"IP":"..."}
curl: (7) Failed to connect to localhost port 1080 after 0 ms: Could not connect to server
❌ Tor connection is  for bridge-3 # curl --socks ... cant connect to the Tor proxy
...
```