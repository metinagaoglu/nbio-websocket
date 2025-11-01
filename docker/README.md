# Docker Deployment Guide

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Build and start
cd docker
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

### Using Docker directly

```bash
# Build image
docker build -f docker/Dockerfile -t nbio-websocket:latest .

# Run container
docker run -d \
  --name nbio-websocket \
  -p 8080:8080 \
  -p 9090:9090 \
  -e WS_LOG_LEVEL=info \
  nbio-websocket:latest

# View logs
docker logs -f nbio-websocket

# Stop and remove
docker stop nbio-websocket
docker rm nbio-websocket
```

## Configuration

### Environment Variables

All configuration is done through environment variables. See `docker-compose.yml` for full list.

**Server Configuration:**
- `WS_HOST` - Listen address (default: 0.0.0.0)
- `WS_PORT` - WebSocket port (default: 8080)

**Client Configuration:**
- `WS_SEND_BUFFER` - Send buffer size (default: 256)
- `WS_PING_INTERVAL` - Ping interval (default: 10s)
- `WS_PONG_TIMEOUT` - Pong timeout (default: 60s)
- `WS_MAX_MESSAGE_SIZE` - Max message size in bytes (default: 1048576)

**Logging:**
- `WS_LOG_LEVEL` - Log level: debug|info|warn|error (default: info)
- `WS_LOG_FORMAT` - Log format: text|json (default: json for Docker)

**Security:**
- `WS_AUTH_ENABLED` - Enable authentication (default: false)
- `WS_BEARER_TOKENS` - Comma-separated tokens (example: token1,token2)
- `WS_RATE_LIMIT_ENABLED` - Enable rate limiting (default: false)
- `WS_RATE_LIMIT_RATE` - Requests per interval (default: 100)
- `WS_RATE_LIMIT_INTERVAL` - Rate limit interval (default: 1m)
- `WS_RATE_LIMIT_BURST` - Burst capacity (default: 10)

### Production Configuration Example

```yaml
version: '3.8'
services:
  websocket:
    image: nbio-websocket:latest
    restart: always
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - WS_LOG_LEVEL=warn
      - WS_LOG_FORMAT=json
      - WS_AUTH_ENABLED=true
      - WS_BEARER_TOKENS=${WS_BEARER_TOKENS}  # from .env file
      - WS_RATE_LIMIT_ENABLED=true
      - WS_RATE_LIMIT_RATE=100
      - WS_RATE_LIMIT_INTERVAL=1m
      - WS_RATE_LIMIT_BURST=10
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:9090/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Endpoints

- **WebSocket**: `ws://localhost:8080/ws`
- **Metrics**: `http://localhost:9090/metrics`
- **Health**: `http://localhost:9090/health`

## Health Checks

The container includes a health check that pings the `/health` endpoint every 30 seconds.

```bash
# Check container health
docker ps

# View health check logs
docker inspect --format='{{json .State.Health}}' nbio-websocket | jq
```

## Monitoring

### Metrics Endpoint

```bash
# Get all metrics
curl http://localhost:9090/metrics | jq

# Example response
{
  "uptime": "1h23m45s",
  "connected_clients": 42,
  "messages_received": 4567,
  "errors_total": 23
}
```

### Optional: Prometheus + Grafana

Uncomment the Prometheus and Grafana services in `docker-compose.yml` to enable monitoring stack.

```bash
# Start with monitoring
docker-compose --profile monitoring up -d

# Access Grafana
open http://localhost:3000
# Default credentials: admin/admin
```

## Logs

```bash
# Follow logs
docker-compose logs -f websocket

# View last 100 lines
docker-compose logs --tail=100 websocket

# JSON log parsing
docker-compose logs websocket | jq -r '.level + " " + .msg'
```

## Troubleshooting

### Container won't start

```bash
# Check logs
docker-compose logs websocket

# Check configuration
docker-compose config

# Validate environment variables
docker-compose exec websocket env | grep WS_
```

### Connection issues

```bash
# Test WebSocket connection
wscat -c ws://localhost:8080/ws

# Test with authentication
wscat -c "ws://localhost:8080/ws" -H "Authorization: Bearer your-token"

# Check if port is accessible
nc -zv localhost 8080
```

### High memory usage

```bash
# Check memory usage
docker stats nbio-websocket

# Set memory limit
docker run -m 512m nbio-websocket:latest
```

## Building Custom Images

### Build with custom version

```bash
docker build \
  -f docker/Dockerfile \
  -t nbio-websocket:v1.2.3 \
  --build-arg VERSION=v1.2.3 \
  .
```

### Multi-arch builds

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f docker/Dockerfile \
  -t nbio-websocket:latest \
  --push \
  .
```

## Security Best Practices

1. **Use secrets for tokens**:
   ```yaml
   environment:
     - WS_BEARER_TOKENS=${WS_BEARER_TOKENS}  # from .env
   ```

2. **Run as non-root** (already configured in Dockerfile)

3. **Enable rate limiting** in production

4. **Use TLS/WSS** with reverse proxy:
   ```yaml
   labels:
     - "traefik.http.routers.websocket.tls=true"
   ```

5. **Regular security updates**:
   ```bash
   docker pull nbio-websocket:latest
   docker-compose up -d
   ```

## Performance Tuning

### For high-traffic scenarios

```yaml
environment:
  - WS_SEND_BUFFER=1024        # Increase buffer
  - WS_MAX_MESSAGE_SIZE=10485760  # 10MB max messages
  - WS_RATE_LIMIT_RATE=1000    # Higher rate limit
  - WS_RATE_LIMIT_BURST=100    # Higher burst

deploy:
  resources:
    limits:
      cpus: '2'
      memory: 2G
    reservations:
      cpus: '1'
      memory: 512M
```

## Deployment Strategies

### Rolling Update

```bash
# Build new image
docker build -f docker/Dockerfile -t nbio-websocket:v2 .

# Update service
docker service update --image nbio-websocket:v2 websocket_service
```

### Blue-Green Deployment

```bash
# Start green (new version)
docker-compose -f docker-compose.green.yml up -d

# Test green
curl http://localhost:8081/health

# Switch traffic (update load balancer)
# ...

# Stop blue (old version)
docker-compose down
```

## Backup and Recovery

### Configuration backup

```bash
# Backup environment config
docker-compose config > backup/config-$(date +%Y%m%d).yml

# Backup .env file
cp .env backup/.env-$(date +%Y%m%d)
```

### Logs backup

```bash
# Export logs
docker-compose logs > backup/logs-$(date +%Y%m%d).log
```

## Support

- **GitHub**: https://github.com/yourusername/nbio-websocket
- **Issues**: https://github.com/yourusername/nbio-websocket/issues