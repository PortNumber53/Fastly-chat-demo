# ⚡ Fastly Chat Demo

A real-time chat application built in Go for **Fastly Compute@Edge**, leveraging **Fastly Fanout** (GRIP) for scalable, edge-delivered real-time message delivery.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![Fastly](https://img.shields.io/badge/Fastly-Compute@Edge-ff2850?logo=fastly)

## Features

- 🚀 **Real-time messaging** via WebSocket connections held by Fastly Fanout
- ⚡ **Fastly Compute@Edge** — runs as a WebAssembly module at the edge
- 🏠 **Multi-room support** with automatic room creation
- 📜 **Message history** per room backed by Fastly KV Store
- 🔌 **REST API** for sending messages and retrieving history
- 🎨 **Modern dark UI** with responsive design for mobile
- 🌍 **Global scale** — Fanout handles millions of concurrent connections

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  Browser Clients                         │
│           (WebSocket / EventSource)                     │
└──────────────┬───────────────────────────┬──────────────┘
               │                           │
               ▼                           ▼
┌─────────────────────────┐    ┌──────────────────────────┐
│    Fastly CDN Edge      │    │   Fastly Compute@Edge    │
│  - Fanout holds WS      │◄───│  - Go Wasm app            │
│  - Publishes messages   │    │  - GRIP control headers   │
│  - Delivers at PoP      │    │  - KV store for history   │
└─────────────────────────┘    └──────────────────────────┘
```

### How It Works

1. Clients connect via WebSocket to a Fastly service with Fanout enabled
2. Fanout forwards the upgrade request to the **Compute@Edge app**
3. The app returns **GRIP headers** (`Grip-Hold: stream`, `Grip-Channel: chat-{room}`)
4. Fanout holds the WebSocket and streams messages from the channel
5. When a message is posted via REST API, the app:
   - Stores it in **Fastly KV Store** for history
   - Writes a **GRIP publish entry** to the `fanout` real-time log endpoint
6. Fanout delivers the message to **all subscribers** on that channel from the nearest edge PoP

## Prerequisites

- [Fastly CLI](https://developer.fastly.com/learning/tools/cli) installed and authenticated
- [TinyGo](https://tinygo.org/getting-started/install/) installed (`tinygo build` target support)
- A Fastly account with **Fanout** enabled on your service
- A **Fastly KV Store** named `chat_state` linked to your service
- A **real-time log endpoint** named `fanout` configured to publish to your Fanout realm

## Quick Start

### Local Development

```bash
# Download dependencies
go mod tidy

# Run tests (skips Fastly host calls)
make test

# Serve locally with Fastly CLI (uses local_server config from fastly.toml)
fastly compute serve
```

### Deploy to Fastly

```bash
# Build and deploy
make deploy

# Or manually:
fastly compute publish
```

## Configuration

All configuration is managed via `fastly.toml` and the Fastly dashboard:

| Resource | Name | Purpose |
|---|---|---|
| KV Store | `chat_state` | Room message history and room list |
| Log Endpoint | `fanout` | GRIP publish entries for Fanout delivery |

### Setting up Fastly Fanout

1. Create a Fastly service with **Fanout** enabled
2. Add a **KV Store** named `chat_state` and link it to your service
3. Add a **real-time log endpoint** named `fanout` pointing to your Fanout GRIP URL
4. Ensure the log endpoint format outputs raw log lines (GRIP JSON entries)

## API Reference

### Health Check
```
GET /api/health
```
Returns service health and deployment mode.

### List Rooms
```
GET /api/rooms
```
Returns known rooms from KV store.

### Room Details & History
```
GET /api/rooms/{roomID}?limit=50
```
Returns room info and recent message history from KV.

### Send a Message
```
POST /api/rooms/{roomID}?username=alice
Content-Type: application/json

{"content": "Hello from the edge!"}
```
Sends a message — stored in KV and published via Fanout log streaming.

### WebSocket (Fanout-held)
```
wss://your-service.fly.dev/ws?room=general&username=alice
```
Connect to a chat room. The connection is held by Fanout at the edge.

## Project Structure

```
.
├── main.go                       # Application entry point (fsthttp handler)
├── go.mod / go.sum               # Go module dependencies
├── fastly.toml                   # Fastly Compute@Edge manifest
├── Makefile                      # Build / test / deploy shortcuts
├── static/
│   ├── index.html                # Chat frontend
│   ├── style.css                 # Dark theme styles
│   └── app.js                    # Client-side WebSocket + REST logic
└── internal/
    ├── grip/
    │   └── grip.go               # GRIP log entry formatting
    ├── models/
    │   └── models.go             # Data models
    └── state/
        └── state.go              # KV-backed room state
```

## Notes on Edge Architecture

- **No long-running server**: Each request is handled by a fresh Compute@Edge instance
- **Stateless real-time**: WebSocket connections are held by Fanout, not the app
- **Message history**: Stored in Fastly KV (eventually consistent; last-write-wins for concurrent updates)
- **User counts**: Not tracked precisely in this demo (would require atomic counters or separate signaling)
- **Sending messages**: Done via REST POST, not WebSocket send, so the Compute app can publish to Fanout

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- [Fastly Compute SDK for Go](https://github.com/fastly/compute-sdk-go)
- [Fastly Fanout](https://developer.fastly.com/reference/fanout/)
- [GRIP Protocol](https://grip.keepthewebweaving.com/)
