# Build stage ──────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

ENV CGO_ENABLED=1

WORKDIR /app

RUN apk add --no-cache --update go gcc g++

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# Runtime stage ────────────────────────────────────────────────────
FROM alpine:3.19
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/server /usr/local/bin/server

# Custom CA certificates support:
# 1. Build-time: COPY your .crt files to /usr/local/share/ca-certificates/
# 2. Run-time: Mount your .crt files as a volume to /usr/local/share/ca-certificates/
# Then run: update-ca-certificates
#
# Example docker-compose volume:
#   volumes:
#     - ./certs:/usr/local/share/ca-certificates:ro
#
# Example entrypoint override:
#   command: sh -c "update-ca-certificates && server"

EXPOSE 8080
ENTRYPOINT ["server"]
