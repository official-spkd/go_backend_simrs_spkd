# syntax=docker/dockerfile:1

ARG GO_VERSION=1.27

# --- build: a static binary, so the runtime image needs no Go and no C libraries
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/simrs-api ./cmd/app

# --- run
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app && adduser -S -G app app

COPY --from=build /out/simrs-api /usr/local/bin/simrs-api

USER app
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${APP_PORT:-8080}/ping" >/dev/null || exit 1

ENTRYPOINT ["simrs-api"]
