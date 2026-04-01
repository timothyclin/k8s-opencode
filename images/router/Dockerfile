FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -o router .

FROM alpine:3.20
COPY --from=builder /app/router /router
EXPOSE 8080
ENTRYPOINT ["/router"]
