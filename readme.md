# Tor Webtunnel Client
A single Docker image to test your [Tor WebTunnel Bridge](https://community.torproject.org/relay/setup/webtunnel/).

The image supports both of these modes with the same entrypoint:
- a single bridge string via `CONNECTION_STRING`
- a file of bridge strings via a mounted file path or `INPUT_FILE`

## Single bridge via environment variable

```bash
docker run --rm -it \
  -e CONNECTION_STRING='webtunnel 10.0.0.161:443 SOMEFINGERPRINT url=https://webtunnel.example.com/your-webtunnel-path ver=0.0.1' \
  webtunnel-client:latest
```

## Multiple bridges via file

```bash
docker run --rm -it \
  -v "$PWD/example-input-file.txt:/app/bridges.txt:ro" \
  webtunnel-client:latest /app/bridges.txt
```

Or via env:

```bash
docker run --rm -it \
  -e INPUT_FILE=/app/bridges.txt \
  -v "$PWD/example-input-file.txt:/app/bridges.txt:ro" \
  webtunnel-client:latest
```

The file format is the same as the example input file, and a trailing `# comment` is treated as the label for the output line.

## Expected output

```text
✅ Tor connection is true for bridge-1 # Check Tor API-Response: {"IsTor":true,"IP":"45.84.107.47"}
❌ Tor connection is false for bridge-2 # Check Tor API-Response: {"IsTor":false,"IP":"..."}
```

## JSON output

Pass `--json` or set `JSON_OUTPUT=1` to get machine-readable output. In JSON mode stdout contains only JSON, and the exit code is `1` if any bridge fails.

Single bridge:

```bash
docker run --rm \
  -e CONNECTION_STRING='webtunnel 10.0.0.161:443 SOMEFINGERPRINT url=https://webtunnel.example.com/your-webtunnel-path ver=0.0.1 # bridge-1' \
  webtunnel-client:latest --json
```

```json
{
  "bridge": "webtunnel 10.0.0.161:443 SOMEFINGERPRINT url=https://webtunnel.example.com/your-webtunnel-path ver=0.0.1",
  "ip": "45.84.107.47",
  "isTor": true,
  "label": "bridge-1",
  "ok": true
}
```

Multiple bridges, results in the order of the file:

```bash
docker run --rm \
  -e JSON_OUTPUT=1 \
  -v "$PWD/example-input-file.txt:/app/bridges.txt:ro" \
  webtunnel-client:latest /app/bridges.txt | jq .
```

```json
[
  {
    "bridge": "webtunnel 10.0.0.161:443 SOMEFINGERPRINT url=https://webtunnel.example.com/your-webtunnel-path ver=0.0.1",
    "ip": "45.84.107.47",
    "isTor": true,
    "label": "bridge-1",
    "ok": true
  },
  {
    "bridge": "webtunnel 10.0.0.161:443 OTHERFINGERPRINT url=https://webtunnel.example.org/other-webtunnel-path ver=0.0.1",
    "error": "could not connect to bridge within 30s: bootstrap at 2% (conn_done_pt), last tor warning: Proxy Client: unable to connect OR connection (handshaking (proxy)) with 10.0.0.161:443 ID=<none> RSA_ID=OTHERFINGERPRINT (\"general SOCKS server failure\")",
    "label": "bridge-2",
    "ok": false
  }
]
```

Failed bridges carry an `error` instead of `ip`/`isTor`. To list only the failing labels:

```bash
docker run --rm -e JSON_OUTPUT=1 \
  -v "$PWD/example-input-file.txt:/app/bridges.txt:ro" \
  webtunnel-client:latest /app/bridges.txt | jq -r '.[] | select(.ok | not) | .label'
```

Slow bridges can take a while to bootstrap. The default limit is 120 seconds per bridge; change it with `BOOTSTRAP_TIMEOUT` (seconds, or a duration such as `90s`).
