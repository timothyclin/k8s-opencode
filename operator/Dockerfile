FROM golang:1.24-alpine AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o manager cmd/main.go

FROM alpine:3.20
RUN adduser -D -u 65532 nonroot
COPY --from=builder /app/manager /manager
USER 65532:65532
ENTRYPOINT ["/manager"]
