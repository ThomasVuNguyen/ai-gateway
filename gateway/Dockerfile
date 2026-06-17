# ── Build stage ──
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy go module files first for better layer caching
COPY go.mod ./
# No go.sum needed (stdlib only, no external deps)

# Copy source code
COPY *.go ./

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o ai-gateway .

# ── Runtime stage ──
FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /build/ai-gateway .

EXPOSE 8080

ENTRYPOINT ["./ai-gateway"]
