# ---- Build Stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /chat-demo .

# ---- Runtime Stage ----
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /chat-demo /app/chat-demo

# Environment defaults
ENV PORT=8080
ENV HOST=0.0.0.0
ENV FASTLY_FANOUT_ENABLED=false

EXPOSE 8080

ENTRYPOINT ["/app/chat-demo"]
