# syntax=docker/dockerfile:1

# Stage 1: Build the Go binary
FROM golang:alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache ca-certificates git

# Cache module dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build static binary without CGO dependencies
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/checkindex-dashboard ./cmd/checkindex-dashboard

# Stage 2: Minimal runtime image
FROM alpine:latest

WORKDIR /app

# Install root certificates, timezone data, and su-exec to safely step down privileges
RUN apk --no-cache add ca-certificates tzdata su-exec

# Create dedicated unprivileged user with UID 1000 matching standard host users
RUN addgroup -g 1000 appgroup && adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# Pre-create configuration directory with appuser permissions
RUN mkdir -p /home/appuser/.checkIndex && chown -R appuser:appgroup /home/appuser && chmod 700 /home/appuser/.checkIndex

# Copy binary from builder and entrypoint
COPY --from=builder /app/checkindex-dashboard /app/checkindex-dashboard
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

# Configure default listener on 0.0.0.0:6767 and 60s background refresh
ENV CHECKINDEX_DASHBOARD_LISTEN_ADDR=0.0.0.0:6767 \
    CHECKINDEX_DASHBOARD_REFRESH_INTERVAL=60s \
    HOME=/home/appuser

EXPOSE 6767

ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["/app/checkindex-dashboard"]
