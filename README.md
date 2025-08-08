# nbio-websocket

This project is a high-performance WebSocket server example using the [nbio](https://github.com/lesismal/nbio) library. It includes a hub structure for client management, broadcasting, json-rpc event routing, and a handler architecture.

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

## Development

- To add a new event, add a new file to the `internal/handlers/` directory and write your function.
- To register the handler to the router, update the `cmd/server.go` file.
- For client management and broadcast operations, check the `internal/hub.go` and `internal/client.go` files.

## Contribution

You can open pull requests and issues.

---

For more information, review the source code or check the [nbio documentation](https://github.com/lesismal/nbio).