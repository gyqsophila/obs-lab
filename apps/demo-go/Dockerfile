
# syntax=docker/dockerfile:1
FROM golang:1.22 as build
WORKDIR /src
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/demo-go ./

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=build /out/demo-go /demo-go
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/demo-go"]
