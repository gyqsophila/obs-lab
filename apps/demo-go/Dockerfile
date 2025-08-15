# syntax=docker/dockerfile:1
FROM golang:latest AS builder

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o demo-go .

FROM tuxmake/base-debian11
WORKDIR /
COPY --from=builder /app/demo-go /demo-go
EXPOSE 8080
USER root
ENTRYPOINT ["/demo-go"]