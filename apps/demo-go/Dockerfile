# syntax=docker/dockerfile:1
FROM golang:latest AS builder

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o demo-go .

FROM alpine:3.18
WORKDIR /
COPY --from=builder /app/demo-go /demo-go
EXPOSE 8080
USER root
ENTRYPOINT ["/demo-go"]