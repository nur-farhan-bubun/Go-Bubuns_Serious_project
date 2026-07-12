FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY shared/ /shared/
COPY chat-service/go.mod chat-service/go.sum ./
RUN go mod download
COPY chat-service/. .
RUN CGO_ENABLED=0 go build -o /chat-service ./cmd/api

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /chat-service /chat-service
EXPOSE 8083
CMD ["/chat-service"]
