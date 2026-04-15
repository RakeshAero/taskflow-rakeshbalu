FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.27.0

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o taskflow ./backend/cmd/api

FROM alpine:3.19 AS runtime

RUN apk add --no-cache ca-certificates postgresql-client

WORKDIR /app

COPY --from=builder /build/taskflow      ./taskflow
COPY --from=builder /build/backend/migrations    ./migrations
COPY --from=builder /build/backend/entrypoint.sh ./entrypoint.sh
COPY --from=builder /build/backend/seed.sql      ./seed.sql
COPY --from=builder /go/bin/goose        /usr/local/bin/goose

RUN sed -i 's/\r$//' ./entrypoint.sh && chmod +x ./entrypoint.sh

EXPOSE 8080

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

ENTRYPOINT ["./entrypoint.sh"]
