# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /build

# Copy go.work and module files for dependency caching
COPY go.work go.work.sum ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY internal/orderintake/go.mod internal/orderintake/go.sum ./internal/orderintake/

# Download dependencies
RUN cd pkg && go mod download
RUN cd internal/orderintake && go mod download

# Copy source code
COPY pkg/ ./pkg/
COPY internal/orderintake/ ./internal/orderintake/
COPY migrations/ ./migrations/
COPY api/ ./api/

# Build the binary
RUN cd internal/orderintake && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/orderintake \
    .

# Runtime stage
FROM alpine:3.19

# Install CA certificates for HTTPS and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -D -u 1000 -G app app

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/orderintake /app/orderintake
COPY --from=builder /build/migrations /app/migrations

# Change ownership
RUN chown -R app:app /app

# Switch to non-root user
USER app

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/orderintake"]
