# ⚡ Fastly Chat Demo

A real-time chat application built in Go, leveraging **Fastly Fanout** (GRIP - Generic Realtime Intermediary Protocol) for scalable real-time message delivery. Also works in **local WebSocket mode** for development and testing.

![Go](https://img.shields.io/badge/Go-1.18+-00ADD8?logo=go)
![Fastly](https://img.shields.io/badge/Fastly-Fanout-ff2850?logo=fastly)
![WebSocket](https://img.shields.io/badge/WebSocket-Gorilla-4CAF50)

## Features

- 🚀 **Real-time messaging** via WebSocket connections
- ⚡ **Fastly Fanout integration** for scalable, edge-delivered real-time updates
- 🏠 **Multi-room support** with automatic room creation/cleanup
- 📜 **Message history** per room with configurable limits
- 🔌 **REST API** for programmatic access and testing
- 🎨 **Modern dark UI** with responsive design for mobile
- 🐳 **Docker-ready** with multi-stage builds
- ⚙️ **Environment-based configuration** with sensible defaults

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                     Browser Clients                       │
│              (WebSocket / EventSource)                    │
└──────────────┬───────────────────────────┬───────────────┘
               │                           │
               ▼                           ▼
┌──────────────────────┐    ┌──────────────────────────────┐
│   Go Chat Server     │    │     Fastly Fanout / GRIP     │
│                      │    │     (optional, for scale)     │
│  ┌────────────────┐  │    │                              │
│  │   WebSocket    │  │    │  - WebSocket proxying        │
│  │     Hub        │──┼───▶│  - HTTP-stream delivery      │
│  └───────┬────────┘  │    │  - Edge-level fan-out        │
│          │           │    └──────────────────────────────┘
│  ┌───────▼────────┐  │
│  │  Fanout        │  │    Without Fanout: Direct WS
│  │  Publisher     │  │    With Fanout:    Publish via GRIP API
│  └────────────────┘  │    ┌──────────────────────────┐
│                      │    │  Fastly CDN Edge          │
│  ┌────────────────┐  │    │  - Subscribers connect    │
│  │   REST API     │  │    │  - Messages delivered at  │
│  └────────────────┘  │    │    the edge               │
│                      │    └──────────────────────────┘
│  ┌────────────────┐  │
│  │  Static Files  │  │
│  │  (embedded)    │  │
│  └────────────────┘  │
└──────────────────────┘
```

### How It Works

**Local Mode (default):** Clients connect directly to the Go server via WebSocket. Messages are broadcast to all connected clients in the same room through the in-memory Hub. This works great for small-scale deployments and development.

**Fastly Fanout Mode:** When Fanout is enabled, messages are additionally published to Fastly's GRIP endpoint. Clients can connect through Fastly's edge network, which handles WebSocket proxying and HTTP-stream delivery at scale. This enables:
- Millions of concurrent connections
- Low-latency delivery from edge PoPs worldwide
- Automatic reconnection handling at the edge
- Reduced load on the origin server

## Quick Start

### Option 1: Run from Source

```bash
# Clone the repository
git clone https://github.com/PortNumber53/Fastly-chat-demo.git
cd Fastly-chat-demo

# Run directly
go run main.go
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

### Option 2: Docker

```bash
# Build and run
docker compose up --build

# Or with Docker directly
docker build -t fastly-chat-demo .
docker run -p 8080:8080 fastly-chat-demo
```

### Option 3: Pre-built Binary

```bash
go build -o chat-demo .
./chat-demo
```

## Configuration

All configuration is via environment variables (see `.env.example`):

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Server listen port |
| `HOST` | `0.0.0.0` | Server listen host |
| `FASTLY_FANOUT_ENABLED` | `false` | Enable Fastly Fanout publishing |
| `FASTLY_GRIP_URL` | - | Fanout GRIP URL (e.g., `http://api.fanout.io/realm/REALM?gs=TOKEN`) |
| `FASTLY_GRIP_KEY` | - | GRIP authentication key (base64-encoded) |
| `FASTLY_FANOUT_DOMAIN` | - | Fanout domain for edge delivery |
| `MAX_ROOMS` | `100` | Maximum concurrent chat rooms |
| `MAX_MESSAGES_PER_ROOM` | `500` | Maximum stored messages per room |
| `MAX_USERNAME_LENGTH` | `32` | Maximum username length |
| `MAX_MESSAGE_LENGTH` | `2000` | Maximum message length |

### CLI Flags

```
./chat-demo -port 9090 -host 127.0.0.1
```

## API Reference

### Health Check
```
GET /api/health
```
Returns service health and active room count.

### List Rooms
```
GET /api/rooms
```
Returns all active rooms with user counts.

### Room Details & History
```
GET /api/rooms/{roomID}?limit=50
```
Returns room info and recent message history.

### Send a Message (REST)
```
POST /api/rooms/{roomID}?username=alice
Content-Type: application/json

{"content": "Hello from the API!"}
```
Send a message to a room via REST (useful for bots and integrations).

### WebSocket
```
ws://localhost:8080/ws?room=general&username=alice
```
Connect to a chat room via WebSocket. Send messages as JSON:
```json
{"content": "Hello everyone!"}
```
Or as plain text strings.

## Fastly Fanout Setup

### Step 1: Create a Fanout Account

1. Sign up at [fastly.com](https://www.fastly.com/products/fanout)
2. Create a Fanout realm in the Fastly dashboard
3. Note your **GRIP URL** and **GRIP Key**

### Step 2: Configure the Server

```bash
export FASTLY_FANOUT_ENABLED=true
export FASTLY_GRIP_URL="http://api.fanout.io/realm/YOUR_REALM?gs=YOUR_TOKEN"
export FASTLY_GRIP_KEY="base64:YOUR_KEY"
export FASTLY_FANOUT_DOMAIN="your-domain.fanoutcdn.com"
```

### Step 3: Configure Fastly Service

In your Fastly service configuration:

1. **Add a Fanout endpoint** pointing to your Go server
2. **Configure GRIP** in your VCL:
```vcl
declare local local.grip.string STRING;
set local.grip.string = "{\"hold_stream\": \"chat-\" + req.url.path}";
```

3. **Enable WebSocket proxying** for the `/ws` path
4. **Set up the log streaming endpoint** named `fanout` for GRIP publish

### How GRIP Works Here

When a message is broadcast:

1. The Go server's Hub sends the message to all **local** WebSocket clients
2. If Fanout is enabled, the Publisher also sends the message to the GRIP API
3. Fastly's edge network delivers the message to **all** subscribers on the channel `chat-{roomID}`
4. Subscribers connected through Fastly receive the message from the nearest edge PoP

## Project Structure

```
.
├── main.go                       # Application entry point
├── go.mod / go.sum               # Go module dependencies
├── Dockerfile                    # Multi-stage Docker build
├── docker-compose.yml            # Docker Compose for easy deployment
├── fastly.toml                   # Fastly Compute@Edge config
├── .env.example                  # Environment variable template
├── static/
│   ├── index.html                # Chat frontend (HTML)
│   ├── style.css                 # Dark theme styles
│   └── app.js                    # Client-side WebSocket logic
├── internal/
│   ├── api/
│   │   └── handlers.go           # REST API handlers
│   ├── hub/
│   │   ├── hub.go                # Chat room hub (core logic)
│   │   └── errors.go             # Domain errors
│   ├── fanout/
│   │   └── publisher.go          # Fastly Fanout / GRIP publisher
│   ├── models/
│   │   └── models.go             # Data models and types
│   └── ws/
│       └── connection.go         # WebSocket connection handling
└── pkg/
    └── config/
        └── config.go             # Configuration management
```

## Development

```bash
# Run with hot reload (install air first)
go install github.com/cosmtrek/air@latest
air

# Run tests
go test ./...

# Format code
go fmt ./...

# Lint
go vet ./...
```

### Adding Features

- **Private rooms**: Add a room password field and validate in JoinRoom
- **User avatars**: Generate deterministic avatar colors from usernames
- **File sharing**: Add multipart upload handler and serve via Fastly Object Store
- **Typing indicators**: Use a lightweight pub/sub channel for typing events
- **Message persistence**: Add a database layer (PostgreSQL, Redis, etc.)

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [Gorilla WebSocket](https://github.com/gorilla/websocket) - Go WebSocket library
- [Fastly Fanout](https://developer.fastly.com/reference/fanout/) - Real-time message delivery at the edge
- [GRIP Protocol](https://grip.keepthewebweaving.com/) - Generic Realtime Intermediary Protocol specification
