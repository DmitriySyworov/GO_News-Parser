FROM golang:1.25.7-alpine AS builder
LABEL org.opencontainers.image.title="Go-Gateway-Swagger"\
      org.opencontainers.image.description="service for collecting documentation of the entire application"\
      org.opencontainers.image.version="1.0.0"\
      org.opencontainers.image.authors="dmitriysyworov1986.com@gmail.com"\
      org.opencontainers.image.source="https://github.com/DmitriySyworov"
WORKDIR /app
COPY shared/go.mod shared/go.sum ./shared/
COPY services/gatewaydocs/go.mod services/gatewaydocs/go.sum ./services/gatewaydocs/
RUN go work init ./shared ./services/gatewaydocs
RUN go mod download -C services/gatewaydocs
COPY services/gatewaydocs/ ./services/gatewaydocs/
COPY shared/ ./shared/
WORKDIR /app/services/gatewaydocs
RUN go build -o /app/services/gatewaydocs/gatewaydocs-app ./cmd/main.go
FROM alpine:3.23.5
WORKDIR /app
COPY --from=builder /app/services/gatewaydocs/gatewaydocs-app .
ENTRYPOINT ["./gatewaydocs-app"]