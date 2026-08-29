FROM golang:1.25.7-alpine AS builder
LABEL org.opencontainers.image.title="Notification-Service"\
      org.opencontainers.image.description="service for sending notifications based on kafka"\
      org.opencontainers.image.version="1.0.0"\
      org.opencontainers.image.authors="dmitriysyworov1986.com@gmail.com"\
      org.opencontainers.image.source="https://github.com/DmitriySyworov"
WORKDIR /app
COPY shared/go.mod shared/go.sum ./shared/
COPY services/notification/go.mod services/notification/go.sum ./services/notification/
RUN go work init ./shared ./services/notification
RUN go mod download -C services/notification
COPY services/notification/ ./services/notification/
COPY shared/ ./shared/
WORKDIR /app/services/notification
RUN go build -o /app/services/notification/notification-app ./cmd/main.go
FROM alpine:3.23.5
WORKDIR /app
COPY --from=builder /app/services/notification/notification-app .
ENTRYPOINT ["./notification-app"]