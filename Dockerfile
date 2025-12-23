FROM golang:1.25 AS builder
WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

# Minimal runtime image
FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /app/server /server

ENV HTTP_PORT=8080 \
    APP_ENV=production

EXPOSE 8080
ENTRYPOINT ["/server"]
