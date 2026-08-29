# syntax=docker/dockerfile:1

FROM golang:1.26.4-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/establishment .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S establishment \
    && adduser -S -G establishment establishment

COPY --from=build /out/establishment /usr/local/bin/establishment

USER establishment:establishment
ENTRYPOINT ["/usr/local/bin/establishment"]
