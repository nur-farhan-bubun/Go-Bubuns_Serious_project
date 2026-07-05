FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /match-service ./cmd/api

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /match-service /match-service
EXPOSE 8082
CMD ["/match-service"]
