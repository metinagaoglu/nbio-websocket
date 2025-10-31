# nbio-websocket

High-performance WebSocket server with horizontal scaling support using the [nbio](https://github.com/lesismal/nbio) library. Features include hub-based client management, JSON-RPC routing, authentication, rate limiting, and distributed broadcasting via Redis/NATS pub/sub.

## Folder Structure

```
nbio-websocket/
├── cmd/                # Main application entry point
│   └── server.go       # Server startup code
├── internal/
│   ├── hub.go          # Hub structure and client management
│   ├── client.go       # Client connection object
│   ├── handler.go      # Event handler interface and router
│   ├── jsonrpc.go      # JSON-RPC message structure and helpers
│   └── handlers/       # Separate handler files for each event
│       ├── broadcast.go
│       └── selfreply.go
├── go.mod
├── go.sum
└── README.md
```

## Installation

1. Install the required modules:

```bash
go mod tidy
```

2. Start the server:

```bash
go run ./cmd/server.go
```

## Architecture

- **Hub:** Manages all client connections and broadcast operations.
- **Client:** An object for each websocket connection.
- **Handler:** Routes incoming json-rpc messages to the relevant handler based on the event field.
- **JSON-RPC:** Messages must be in json-rpc 2.0 format.

## JSON-RPC Message Examples

### Broadcast Event
```json
{
  "jsonrpc": "2.0",
  "method": "broadcast",
  "params": { "text": "Message to everyone!" },
  "id": 1
}
```
The message sent by the client is delivered to all connected clients.

### Self Reply Event
```json
{
  "jsonrpc": "2.0",
  "method": "self.reply",
  "params": { "text": "Reply only to me!" },
  "id": 2
}
```
The client that sends this message receives a reply only to itself (echo).

## Horizontal Scaling

The server supports horizontal scaling across multiple instances using pub/sub adapters for distributed message broadcasting.

### Architecture

When multiple server instances are running:
1. A client connects to any instance via load balancer
2. When that client sends a broadcast message, the instance publishes it to the pub/sub channel
3. All instances (including the sender) receive the message from pub/sub
4. Each instance broadcasts the message to its local connected clients

This ensures messages reach all clients across all instances without direct inter-instance communication.

### Supported Adapters

**Local** (default): Single instance mode, no pub/sub
```bash
export WS_PUBSUB_ENABLED=false
export WS_PUBSUB_ADAPTER=local
```

**Redis**: Recommended for production horizontal scaling
```bash
export WS_PUBSUB_ENABLED=true
export WS_PUBSUB_ADAPTER=redis
export WS_REDIS_URL=redis://localhost:6379/0
export WS_PUBSUB_CHANNEL=websocket.broadcast
```

**NATS**: High-performance messaging system for cloud-native deployments
```bash
export WS_PUBSUB_ENABLED=true
export WS_PUBSUB_ADAPTER=nats
export WS_NATS_URL=nats://localhost:4222
export WS_PUBSUB_CHANNEL=websocket.broadcast
```

### Quick Start - Multiple Instances

#### Option 1: Manual Launch
```bash
# Terminal 1: Start Redis
docker run -d --name redis -p 6379:6379 redis:7-alpine

# Terminal 2: Instance 1
WS_PUBSUB_ENABLED=true \
WS_PUBSUB_ADAPTER=redis \
WS_REDIS_URL=redis://localhost:6379/0 \
WS_PORT=8080 \
./bin/nbio-ws

# Terminal 3: Instance 2
WS_PUBSUB_ENABLED=true \
WS_PUBSUB_ADAPTER=redis \
WS_REDIS_URL=redis://localhost:6379/0 \
WS_PORT=8081 \
./bin/nbio-ws

# Terminal 4: Instance 3
WS_PUBSUB_ENABLED=true \
WS_PUBSUB_ADAPTER=redis \
WS_REDIS_URL=redis://localhost:6379/0 \
WS_PORT=8082 \
./bin/nbio-ws
```

#### Option 2: Docker Compose
```bash
cd docker
docker-compose -f docker-compose.scale.yml up -d redis
docker-compose -f docker-compose.scale.yml up -d --scale websocket=5
```

This starts 5 WebSocket instances behind nginx load balancer with Redis pub/sub.

### Testing Distributed Broadcasting

```bash
# Terminal 1: Connect to instance 1
wscat -c ws://localhost:8080/ws

# Terminal 2: Connect to instance 2
wscat -c ws://localhost:8081/ws

# Terminal 3: Connect to instance 3
wscat -c ws://localhost:8082/ws

# Send message from any terminal
{"jsonrpc":"2.0","method":"broadcast","params":{"text":"Hello from distributed system!"},"id":1}

# All terminals receive the message regardless of which instance they're connected to
```

### Load Balancing Strategies

**Round Robin** (nginx default):
```nginx
upstream websocket {
    server websocket1:8080;
    server websocket2:8080;
    server websocket3:8080;
}
```

**IP Hash** (sticky sessions):
```nginx
upstream websocket {
    ip_hash;
    server websocket1:8080;
    server websocket2:8080;
    server websocket3:8080;
}
```

**Least Connections**:
```nginx
upstream websocket {
    least_conn;
    server websocket1:8080;
    server websocket2:8080;
    server websocket3:8080;
}
```

### Monitoring Multiple Instances

```bash
# Check health of all instances
curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health

# Aggregate metrics
curl http://localhost:8080/metrics
curl http://localhost:8081/metrics
curl http://localhost:8082/metrics

# With Docker Compose
docker-compose -f docker-compose.scale.yml ps
docker-compose -f docker-compose.scale.yml logs -f websocket
```

### Performance Considerations

**Redis Pub/Sub**:
- Latency: ~1-2ms additional overhead
- Throughput: 100K+ messages/sec
- Memory: Minimal (~10MB for Redis)
- Best for: < 10 instances, simple deployments

**NATS Pub/Sub**:
- Latency: <1ms additional overhead
- Throughput: 1M+ messages/sec
- Memory: Very low (~5MB for NATS)
- Best for: 10+ instances, cloud-native, high throughput

**Scaling Limits**:
- Single instance: 10K concurrent connections
- With Redis: 50K+ connections (5+ instances)
- With NATS: 100K+ connections (10+ instances)
- Load balancer becomes bottleneck beyond 100K connections

### Production Deployment

```yaml
# docker-compose.yml
version: '3.8'
services:
  redis:
    image: redis:7-alpine
    restart: always

  websocket:
    image: nbio-websocket:latest
    environment:
      WS_PUBSUB_ENABLED: "true"
      WS_PUBSUB_ADAPTER: redis
      WS_REDIS_URL: redis://redis:6379/0
      WS_AUTH_ENABLED: "true"
      WS_BEARER_TOKENS: ${SECRET_TOKEN}
      WS_RATE_LIMIT_ENABLED: "true"
    depends_on:
      - redis
    deploy:
      replicas: 5

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    depends_on:
      - websocket
```

### Troubleshooting

**Messages not reaching all instances:**
- Verify pub/sub connection: Check logs for "PubSub adapter configured"
- Test pub/sub directly: Use redis-cli or nats-cli to verify connectivity
- Check channel name: Ensure all instances use same `WS_PUBSUB_CHANNEL`

**High latency with scaling:**
- Monitor Redis/NATS latency: Add metrics for pub/sub operations
- Check network: Use same data center/region for all instances
- Optimize Redis: Use `redis.conf` tuning for persistence=off in cache scenarios

**Connection distribution issues:**
- Verify load balancer health checks
- Check sticky session configuration if needed
- Monitor instance CPU/memory for overload

## Development

- To add a new event, add a new file to the `internal/handlers/` directory and write your function.
- To register the handler to the router, update the `cmd/server.go` file.
- For client management and broadcast operations, check the `internal/hub.go` and `internal/client.go` files.
- For scaling configuration, see `internal/config/config.go` and `internal/pubsub/` directory.

## Documentation

- **Quick Start**: See `QUICKSTART.md`
- **Scaling Guide**: See `claudedocs/SCALING-GUIDE.md`
- **Docker Guide**: See `docker/README.md`
- **Complete Documentation**: See `claudedocs/` directory

## Contribution

You can open pull requests and issues.

---

For more information, review the source code or check the [nbio documentation](https://github.com/lesismal/nbio).