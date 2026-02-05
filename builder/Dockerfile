FROM golang:1.25-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY agent/ ./agent/
RUN CGO_ENABLED=0 go build -o /builder ./agent/

FROM alpine:3.21
RUN apk add --no-cache git openssh-client ca-certificates

COPY --from=build /builder /builder

ENTRYPOINT ["/builder"]
