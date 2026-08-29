# syntax=docker/dockerfile:1

FROM golang:1.26.4-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/kitchen \
    ./cmd/kitchen

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S kitchen \
    && adduser -S -G kitchen kitchen

COPY --from=build /out/kitchen /usr/local/bin/kitchen

USER kitchen:kitchen
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/kitchen"]
