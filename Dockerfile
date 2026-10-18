FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o webhookserver main.go


FROM alpine:3.24
WORKDIR /app
COPY --from=builder /app/webhookserver .
USER 65534
EXPOSE 8000
ENTRYPOINT ["/app/webhookserver"]
