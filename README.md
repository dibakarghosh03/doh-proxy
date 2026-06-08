# doh-proxy

A lightweight DNS-over-HTTPS (DoH) proxy written in Go from scratch.

Listens for plain DNS queries over UDP on localhost, forwards them to an upstream DoH resolver (Cloudflare `1.1.1.1`) over HTTPS, caches responses using each answer's TTL, and returns the result to the caller.

---

## How it works

```
Your OS / App
     │
     │  UDP DNS query (port 5353)
     ▼
doh-proxy (localhost)
     │
     ├── cache hit?  ──yes──▶  return cached response (0ms)
     │
     no
     │
     │  HTTPS POST (application/dns-message)
     ▼
Cloudflare DoH (cloudflare-dns.com)
     │
     ▼
doh-proxy caches response with TTL
     │
     ▼
Your OS / App gets the answer
```

---

## Features

- **DNS wire format parser** — parses headers, questions, and answer records including compression pointers
- **UDP listener** — handles concurrent queries, one goroutine per request
- **DoH forwarding** — forwards raw DNS wire format over HTTPS to Cloudflare
- **TTL-aware cache** — caches responses and expires them based on the minimum TTL across all answer records
- **Background cache cleanup** — periodic sweep evicts stale entries from memory
- **ID patching** — correctly rewrites response IDs to match each client's query

---

## Project structure

```
doh-proxy/
├── main.go        — entry point
├── dns.go         — DNS wire format parser and structs
├── listener.go    — UDP listener and query handler
├── resolver.go    — DoH forwarding to upstream resolver
├── cache.go       — TTL-aware in-memory cache
└── go.mod
```

---

## Getting started

### Prerequisites

- Go 1.20 or higher

### Run

```bash
git clone https://github.com/dibakarghosh03/doh-proxy
cd doh-proxy
go run .
```

The proxy listens on `127.0.0.1:5353` by default.

### Test

```bash
dig @127.0.0.1 -p 5353 google.com
dig @127.0.0.1 -p 5353 github.com
```

Run the same query twice — the second should return in `0ms` from cache.

---

## Use as system DNS resolver (Linux)

**1. Build the binary and grant port 53 access:**

```bash
go build -o doh-proxy .
sudo setcap cap_net_bind_service=+ep ./doh-proxy
```

**2. Change the listen address in `main.go`:**

```go
log.Fatal(startUDPListener("127.0.0.1:53"))
```

**3. Disable systemd-resolved (Ubuntu):**

```bash
sudo systemctl stop systemd-resolved
sudo systemctl disable systemd-resolved
```

**4. Point your system at the proxy:**

```bash
# /etc/resolv.conf
nameserver 127.0.0.1
```

Every DNS lookup on your machine now goes through doh-proxy.

**To revert:**

```bash
sudo systemctl enable systemd-resolved
sudo systemctl start systemd-resolved
# restore /etc/resolv.conf to: nameserver 127.0.0.53
```

---

## Configuration

Currently configured via constants in source. Defaults:

| Setting | Value |
|---|---|
| Listen address | `127.0.0.1:5353` |
| Upstream resolver | `https://cloudflare-dns.com/dns-query` |
| Cache cleanup interval | `1 minute` |

---

## Possible extensions

- [ ] Metrics — cache hit/miss rate, query counts per domain
- [ ] Multiple upstream resolvers with fallback
- [ ] Config file or CLI flags for listen address and upstream URL
- [ ] NXDOMAIN caching
- [ ] DNS over TCP support (for responses > 512 bytes)

---
