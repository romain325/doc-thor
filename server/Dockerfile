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

EXPOSE 8080
ENTRYPOINT ["server"]
