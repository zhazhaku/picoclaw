# Reef Deployment Guide

This guide covers deploying Reef in various configurations, from single-machine testing to multi-node production setups.

## Table of Contents

- [Single Machine (Development)](#single-machine-development)
- [Docker Compose](#docker-compose)
- [Systemd Services](#systemd-services)
- [Multi-Node Cluster](#multi-node-cluster)
- [Reverse Proxy (Nginx)](#reverse-proxy-nginx)
- [Security Hardening](#security-hardening)

## Single Machine (Development)

Run Server and Client on the same machine for local testing:

```bash
# Terminal 1: Start Server
picoclaw reef-server --ws-addr :8080 --admin-addr :8081

# Terminal 2: Start Client
picoclaw --config client-config.json
```

`client-config.json`:

```json
{
  "providers": {
    "openai": {
      "api_key": "sk-..."
    }
  },
  "channels": {
    "swarm": {
      "enabled": true,
      "mode": "client",
      "server_url": "ws://localhost:8080",
      "role": "coder",
      "skills": ["github", "write_file"],
      "capacity": 3
    }
  }
}
```

## Docker Compose

A ready-to-use `docker-compose.reef.yml` is provided in the `docker/` directory with pre-configured Server and Client configs.

```bash
# Start the full Reef cluster (Server + 2 Clients)
cd docker
docker compose -f docker-compose.reef.yml up -d

# Check status
docker compose -f docker-compose.reef.yml ps

# View logs
docker compose -f docker-compose.reef.yml logs -f

# Stop
docker compose -f docker-compose.reef.yml down
```

The compose file includes:
- **reef-server** — Reef Server with `mode: "server"` config
- **reef-client-coder** — Coder role client with skills: github, write_file, exec, read_file, edit_file
- **reef-client-analyst** — Analyst role client with skills: web_fetch, web_search, summarize, read_file

Config files are in `docker/`:
- `reef-server-config.json`
- `reef-client-coder-config.json`
- `reef-client-analyst-config.json`

To customize, edit the config JSON files or set environment variables:

```bash
OPENAI_API_KEY=sk-... REEF_TOKEN=my-secret docker compose -f docker-compose.reef.yml up -d
```

Start:

```bash
docker compose up -d
```

## Systemd Services

### Reef Server

`/etc/systemd/system/reef-server.service`:

```ini
[Unit]
Description=Reef Server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/picoclaw reef-server --ws-addr :8080 --admin-addr :8081 --token /etc/reef/token.txt
Restart=always
RestartSec=5
User=reef
Group=reef

[Install]
WantedBy=multi-user.target
```

### Reef Client

`/etc/systemd/system/reef-client.service`:

```ini
[Unit]
Description=Reef Client
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/picoclaw --config /etc/reef/client-config.json
Restart=always
RestartSec=10
User=reef
Group=reef

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now reef-server
sudo systemctl enable --now reef-client
```

## Multi-Node Cluster

For a production cluster with multiple Server and Client nodes:

```
┌─────────────────┐
│  Load Balancer  │  (Nginx / Traefik)
│   :80 / :443    │
└────────┬────────┘
         │
    ┌────┴────┐
    ▼         ▼
┌────────┐ ┌────────┐
│Server 1│ │Server 2│  (HAProxy / Keepalived)
│:8080   │ │:8080   │
└───┬────┘ └───┬────┘
    │          │
    └────┬─────┘
         │ WebSocket
    ┌────┴────┐
    ▼         ▼
┌────────┐ ┌────────┐
│Client A│ │Client B│
│coder   │ │analyst │
└────────┘ └────────┘
```

**Note**: v1 uses in-memory state. For true HA, you need:
- Sticky sessions (same client always connects to same server)
- Or shared state backend (planned for v2)

## Reverse Proxy (Nginx)

```nginx
upstream reef_ws {
    server 127.0.0.1:8080;
}

upstream reef_admin {
    server 127.0.0.1:8081;
}

server {
    listen 80;
    server_name reef.example.com;

    location /ws {
        proxy_pass http://reef_ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
    }

    location /admin/ {
        proxy_pass http://reef_admin;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /tasks {
        proxy_pass http://reef_admin;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Security Hardening

### Token Authentication

Always set a token in production:

```bash
picoclaw reef-server --token "$(openssl rand -hex 32)"
```

Store the same token in client configs.

### TLS

For production, terminate TLS at Nginx or use a reverse proxy. Native TLS support is planned for v2.

### Firewall Rules

```bash
# Allow only trusted clients to connect to WebSocket
sudo ufw allow from 10.0.0.0/8 to any port 8080
sudo ufw allow from 10.0.0.0/8 to any port 8081

# Or bind to internal interface only
picoclaw reef-server --ws-addr 10.0.0.1:8080 --admin-addr 10.0.0.1:8081
```

### Rate Limiting

Use Nginx or a WAF to rate-limit:
- `/tasks` endpoint (task submission)
- `/ws` endpoint (WebSocket connections)

Example with Nginx limit_req:

```nginx
limit_req_zone $binary_remote_addr zone=tasks:10m rate=10r/s;

location /tasks {
    limit_req zone=tasks burst=20 nodelay;
    proxy_pass http://reef_admin;
}
```
