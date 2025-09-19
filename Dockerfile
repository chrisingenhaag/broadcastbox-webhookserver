FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o webhookserver main.go


FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/webhookserver .
EXPOSE 8000
ENTRYPOINT ["/app/webhookserver"]
