FROM golang:1.23-alpine AS builder
WORKDIR /build

ENV GOTOOLCHAIN=auto

RUN apk add --no-cache git ca-certificates

COPY shared/ ./shared/
COPY billing_service/ ./billing_service/


WORKDIR /build/billing_service


RUN go mod download


RUN CGO_ENABLED=0 GOOS=linux go build -o /build/billing_service_bin ./cmd/api/main.go

FROM alpine:latest
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /build/billing_service_bin ./billing_service
COPY --from=builder /build/billing_service/.env ./

RUN chmod +x billing_service

CMD ["./billing_service"]