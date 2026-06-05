# Stage 1: Build Go binary
FROM golang:1.25-bookworm AS builder

WORKDIR /app

# Install gcc for CGo (SQLite)
RUN apt-get update && apt-get install -y gcc musl-dev && rm -rf /var/lib/apt/lists/*

# Cache Go module downloads
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download

# Copy backend source
COPY backend/ ./backend/

# Build statically-linked binary with CGo enabled for SQLite
RUN cd backend && CGO_ENABLED=1 go build -o /sluff-server ./cmd/sluff-server

# Stage 2: Runtime
FROM debian:bookworm-slim AS runtime

WORKDIR /app

# Install ca-certificates, curl, and Litestream
RUN apt-get update && apt-get install -y ca-certificates curl && rm -rf /var/lib/apt/lists/*

# Install Litestream
RUN curl -fsSL https://github.com/benbjohnson/litestream/releases/download/v0.3.13/litestream-v0.3.13-linux-amd64.deb -o /tmp/litestream.deb && \
    dpkg -i /tmp/litestream.deb && \
    rm /tmp/litestream.deb

# Copy binary from builder
COPY --from=builder /sluff-server /app/sluff-server

# Copy Litestream config and entrypoint
COPY litestream.yml /app/litestream.yml
COPY entrypoint.sh /app/entrypoint.sh

# Create data directory for SQLite
RUN mkdir -p /app/data

# Create non-root user
RUN useradd --create-home --shell /bin/bash appuser && \
    chown -R appuser:appuser /app
USER appuser

# Bind all interfaces inside the container; the container boundary (and
# published ports) is the isolation, not the listen address. The server
# defaults to 127.0.0.1 for host-OS runs (`make dev`).
ENV HOST=0.0.0.0

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

EXPOSE 8080

CMD ["/app/entrypoint.sh"]
