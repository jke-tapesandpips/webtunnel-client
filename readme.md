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
docker run --rm -it <your-tag> test-webtunnel-client.sh example-input-file.txt
```
Or you can use the prebuilt image:
```
docker run --rm -it bubatzlegal/webtunnel-client:automated-latest test-webtunnel-client.sh example-input-file.txt
```

Example Output:
```
✅ Tor connection is true for bridge-1
❌ Tor connection is false for bridge-2
✅ Tor connection is true for bridge-3
✅ Tor connection is true for bridge-4
```