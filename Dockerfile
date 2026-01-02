FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -ldflags="-s -w" -o hlg ./cmd/hlg

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /data
COPY --from=builder /app/hlg /usr/local/bin/
EXPOSE 8080
CMD ["hlg", "--db", "/data/hlg.db"]
