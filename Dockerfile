# ==========================================
# Build Stage
# ==========================================
FROM golang:alpine AS builder

# Install git, build tools and ca-certificates
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /workspace

# Leverage Docker layer caching for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build static binary with CGO disabled
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o /workspace/bin/mediahelper \
    ./cmd/mediahelper

# ==========================================
# Final Runtime Stage
# ==========================================
FROM alpine:latest

# Install runtime dependencies (ca-certificates for HTTPS calls, tzdata for timezone configuration)
RUN apk add --no-cache ca-certificates tzdata \
    && update-ca-certificates

# Set default timezone to UTC (can be overridden via -e TZ=Asia/Shanghai)
ENV TZ=Asia/Shanghai

WORKDIR /app

# Create necessary directories for storage, configs, and media mounts
RUN mkdir -p /app/configs /app/data /media /downloads

# Copy compiled binary from builder
COPY --from=builder /workspace/bin/mediahelper /app/mediahelper

# Copy default configuration file as fallback
COPY configs/config.json /app/configs/config.json.default

# Ensure config exists on first boot if not mounted
RUN cp /app/configs/config.json.default /app/configs/config.json

# Declare mountable volumes
VOLUME ["/app/data", "/app/configs", "/media", "/downloads"]

# Expose default HTTP API & Webhook port
EXPOSE 9090

# Default entrypoint and flags
ENTRYPOINT ["/app/mediahelper"]
CMD ["-config", "/app/configs/config.json"]

