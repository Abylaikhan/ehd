# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS base
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum* ./
RUN go mod download

# --- dev: hot-reload через air, исходники монтируются volume'ом ---
FROM base AS dev
RUN go install github.com/air-verse/air@latest
EXPOSE 8080
CMD ["sh", "-c", "go mod tidy && air -c .air.toml"]

# --- builder ---
FROM base AS builder
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/ehd-api ./cmd/ehd-api

# --- prod: минимальный distroless, non-root ---
FROM gcr.io/distroless/static-debian12:nonroot AS prod
COPY --from=builder /bin/ehd-api /ehd-api
EXPOSE 8080
ENTRYPOINT ["/ehd-api"]
