FROM golang:1.24-alpine AS builder
WORKDIR /app

# Install CA certificates for TLS connections
RUN apk add --no-cache ca-certificates

COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /product-api

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /product-api .
COPY wait-for.sh .
RUN chmod +x wait-for.sh
CMD ["./product-api"]