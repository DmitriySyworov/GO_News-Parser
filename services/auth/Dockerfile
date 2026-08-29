FROM golang:1.25.7-alpine AS builder
LABEL org.opencontainers.image.title="Auth-User-Service"\
      org.opencontainers.image.description="service for authorizing users and performing actions on user accounts"\
      org.opencontainers.image.version="1.0.0"\
      org.opencontainers.image.authors="dmitriysyworov1986.com@gmail.com"\
      org.opencontainers.image.source="https://github.com/DmitriySyworov"
WORKDIR /app
COPY shared/go.mod shared/go.sum ./shared/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/
RUN go work init ./shared ./services/auth
RUN go mod download -C services/auth
COPY services/auth/ ./services/auth/
COPY shared/ ./shared/
WORKDIR /app/services/auth
RUN go build -o /app/services/auth/auth-app ./cmd/app/main.go
RUN go build -o /app/services/auth/croncleaner ./cmd/croncleaner/cleaner.go
WORKDIR /app/services/auth/cmd/migration
RUN go build -o /app/services/auth/migrator .
FROM alpine:3.23.5
WORKDIR /app
COPY --from=builder /app/services/auth/auth-app .
COPY --from=builder /app/services/auth/migrator .
COPY --from=builder /app/services/auth/croncleaner .
ENTRYPOINT ["./auth-app"]