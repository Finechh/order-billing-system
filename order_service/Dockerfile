

FROM golang:1.23-alpine AS builder
WORKDIR /build

ENV GOTOOLCHAIN=auto

RUN apk add --no-cache git ca-certificates


COPY shared/ ./shared/
COPY order_service/ ./order_service/


WORKDIR /build/order_service


RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /build/order_service_bin ./cmd/api/main.go


FROM alpine:latest
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /build/order_service_bin ./order_service
COPY --from=builder /build/order_service/.env ./

RUN chmod +x order_service

EXPOSE 50051
CMD ["./order_service"]