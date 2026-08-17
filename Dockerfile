# Multi-stage Dockerfile for the public-facility preventive maintenance
# collaboration platform. Builds the Go server, then layers in the compiled
# Vue frontend assets so a single container can serve both.

# ---------- stage 1: build frontend ----------
FROM node:20-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm ci --no-audit --no-fund || npm install --no-audit --no-fund
COPY web/ ./
RUN npm run build

# ---------- stage 2: build Go server ----------
FROM golang:1.24-alpine AS go-builder
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/baseline-server ./cmd/server

# ---------- stage 3: runtime ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=go-builder /out/baseline-server /app/baseline-server
COPY --from=web-builder /web/dist /app/web/dist
COPY migrations /app/migrations
COPY api /app/api
RUN mkdir -p /app/var/attachments && chown -R app:app /app
USER app
ENV HTTP_ADDR=:8080 \
    HTTP_MODE=release \
    USE_MEMORY_STORE=true \
    SEED_ON_START=true \
    STORAGE_ROOT=/app/var/attachments
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=2s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/baseline-server"]
