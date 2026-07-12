FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY shared/ /shared/
COPY user-service/go.mod user-service/go.sum ./
RUN go mod download
COPY user-service/. .
RUN CGO_ENABLED=0 go build -o /user-service ./cmd/api

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /user-service /user-service
EXPOSE 8081
CMD ["/user-service"]
