FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o service-b

FROM alpine:3.19

# Install wget for health checks
RUN apk add --no-cache wget

WORKDIR /app
COPY --from=builder /app/service-b .

EXPOSE 8081
CMD ["./service-b"]
