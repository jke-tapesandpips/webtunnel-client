# Tor Webtunnel Client
A docker container to test your webtunnel "bridges"

Usage:
```
docker build . -t <your-tag>
docker run -it -e CONNECTION_STRING="webtunnel 10.0.0.161:443 50M3F1N93RPR1N7 url=https://webtunnel.example.com/your-webtunnel-path ver=0.0.1" <your-tag>
```
Or you can use the prbuilt image:
```
docker run -it -e CONNECTION_STRING="webtunnel 10.0.0.161:443 50M3F1N93RPR1N7 url=https://webtunnel.example.com/your-webtunnel-path ver=0.0.1" bubatzlegal/webtunnel-client:latest
```