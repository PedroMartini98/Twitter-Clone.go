FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o twitter-clone ./cmd/server/

FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/twitter-clone .

COPY assets/ ./assets/

# Expose the port the app runs on
EXPOSE 8080

CMD ["./twitter-clone"]
