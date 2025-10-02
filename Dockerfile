FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build binary with project name
RUN CGO_ENABLED=0 GOOS=linux go build -o debank ./cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/debank .

EXPOSE 8080
CMD ["./debank"]
