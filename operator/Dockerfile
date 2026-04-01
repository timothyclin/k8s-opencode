FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o manager cmd/main.go

FROM alpine:3.20
COPY --from=builder /app/manager /manager
ENTRYPOINT ["/manager"]
