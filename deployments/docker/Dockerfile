# Production Dockerfile - Multi-stage build with SQLite support
FROM golang:1.23-alpine AS builder

# Install build dependencies including C libraries for SQLite
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s' \
    -a \
    -o ping-monitor .

# Production stage - use Alpine instead of scratch for SQLite libraries
FROM alpine:latest

# Install runtime dependencies for SQLite
RUN apk add --no-cache ca-certificates tzdata sqlite

# Copy our binary
COPY --from=builder /app/ping-monitor /ping-monitor

# Copy default config files
COPY configs/config.yaml /app/config.yaml
COPY configs/sites.yaml /app/sites.yaml

# Copy web templates and static files
COPY web/ /app/web/

# Create data directory for SQLite database
RUN mkdir -p /app/data

# Set working directory
WORKDIR /app

# Expose port
EXPOSE 8080

# Add health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/ping-monitor", "--health-check"]

# Run the binary
ENTRYPOINT ["/ping-monitor"]
