# Order Billing System

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go&logoColor=white)
![gRPC](https://img.shields.io/badge/gRPC-Protobuf-4285F4?style=flat&logo=google&logoColor=white)
![Kafka](https://img.shields.io/badge/Apache_Kafka-231F20?style=flat&logo=apachekafka&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?style=flat&logo=postgresql&logoColor=white)
![Prometheus](https://img.shields.io/badge/Prometheus-E6522C?style=flat&logo=prometheus&logoColor=white)
![Grafana](https://img.shields.io/badge/Grafana-F46800?style=flat&logo=grafana&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=flat&logo=docker&logoColor=white)
![order_service coverage](https://img.shields.io/badge/order__service%20coverage-63.9%25-yellow?style=flat&logo=go)
![billing_service coverage](https://img.shields.io/badge/billing__service%20coverage-88.5%25-brightgreen?style=flat&logo=go)

A production-ready microservices system built in Go that handles order management and billing. Demonstrates real-world backend engineering patterns: Outbox Pattern, idempotent Kafka consumers, gRPC APIs, and full observability.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Client (gRPC)                           │
└───────────────┬─────────────────────────┬───────────────────────┘
                │                         │
                ▼                         ▼
   ┌────────────────────┐     ┌─────────────────────┐
   │   order_service    │     │   billing_service   │
   │   :50051 (gRPC)    │     │   :50052 (gRPC)     │
   │   :9094 (metrics)  │     │   :9093 (metrics)   │
   └────────┬───────────┘     └──────────┬──────────┘
            │                            │
            │  Outbox Pattern            │  Kafka Consumer
            ▼                            │  (idempotent)
   ┌────────────────┐                    │
   │  postgres_order│                    │
   │  outbox_events │──── Relay ────► Kafka ◄──────────┘
   │  orders        │    (polling)   3 topics
   └────────────────┘                   
                            ┌───────────────────┐
                            │  postgres_billing  │
                            │  invoices          │
                            │  processed_events  │
                            └───────────────────┘

         Prometheus (:9090) + Grafana (:3000)
```

## Key Engineering Decisions

### Outbox Pattern
Order events (Created/Paid/Cancelled) are written atomically to the `outbox_events` table in the same transaction as the order itself. A background relay polls the table and publishes to Kafka. This guarantees **no event is ever lost** even if Kafka is temporarily unavailable.

### Idempotent Kafka Consumer
Each processed event is stored in `processed_events` with a business key `{EventType}-{OrderID}` (e.g. `OrderCreated-550e8400-...`). On duplicate delivery the consumer skips processing and commits the offset. The key is stable across restarts, database wipes, and Kafka offset resets — unlike offset-based keys which cause silent data loss after redeployment.

`Save` is called **before** `CommitMessages`. If the service crashes between the two, the message is redelivered but the idempotency check rejects it. This ensures at-least-once delivery with safe deduplication.

### Dead Letter Queue
Messages that fail all retry attempts are published to `billing-dlq-topic` with the original payload, error reason, and timestamp. This prevents poison messages from blocking the consumer indefinitely.

### Retry with Exponential Backoff + Jitter
Failed message processing is retried up to 3 times with exponential backoff and random jitter to avoid thundering herd on partial outages.

### Panic Recovery
Both gRPC handlers and Kafka message handlers are wrapped with panic recovery middleware. A panic in one message does not crash the service.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.25 |
| Transport | gRPC + Protobuf |
| Message Broker | Apache Kafka (segmentio/kafka-go) |
| Database | PostgreSQL 15 (GORM) |
| Observability | Prometheus + Grafana |
| Testing | testify + gomock |
| Config | godotenv |

## Project Structure

```
order-billing-system/
├── proto/                  # Source .proto files
│   ├── order/v1/
│   └── billing/v1/
│
├── order_service/
│   ├── cmd/api/            # Entrypoint
│   └── internal/
│       ├── app/            # Wiring (DB, Kafka, gRPC server)
│       ├── config/         # Env config
│       ├── domain/         # Order model, events, statuses
│       ├── service/        # Business logic + validation
│       │   └── mocks/      # Testify mocks
│       ├── repository/     # PostgreSQL via GORM
│       ├── outbox/         # Outbox model, repo, relay
│       └── grpc/           # gRPC server + proto mapping
│
├── billing_service/
│   ├── cmd/api/            # Entrypoint
│   └── internal/
│       ├── app/            # Wiring
│       ├── config/         # Env config
│       ├── billing_domain/ # Invoice model, events, statuses
│       ├── service/        # Invoice lifecycle logic
│       │   └── mocks/      # Testify mocks
│       ├── repository/     # Write repo + read repo (CQRS-lite)
│       ├── query/          # Read-side query service
│       ├── kafka/
│       │   ├── consumer/   # Idempotent consumer
│       │   ├── handler/    # Per-topic message routing
│       │   └── reliability/# Retry, DLQ producer
│       └── grpc/           # gRPC query server
│
└── shared/
    ├── errors/             # Typed AppError (code + status)
    ├── logger/             # Structured JSON logger with request_id
    ├── middleware/         # gRPC interceptors (requestid, validation, recovery)
    ├── middleware_grpc/    # Metrics interceptor
    ├── money/              # Money value type with DB serialization
    ├── currency/           # Static USD converter (USD/EUR/RUB)
    ├── observability/      # Prometheus metrics (gRPC, Kafka, DB, business)
    └── requestid/          # Request ID context propagation
```

## gRPC API

### OrderService — `:50051`

| Method | Request | Description |
|--------|---------|-------------|
| `CreateOrder` | `items[]` with product_id, quantity, price | Creates order, converts all prices to USD, publishes OrderCreated event |
| `GetOrder` | `order_id` | Returns order with current status |
| `MarkOrderPaid` | `order_id` | Transitions CREATED → PAID, publishes OrderPaid event |
| `CancelOrder` | `order_id` | Transitions CREATED → CANCELLED (idempotent), publishes OrderCancelled event |

**Supported currencies:** USD, EUR, RUB (converted to USD at order creation)

**Order state machine:**
```
CREATED ──► PAID
   │
   └──────► CANCELLED
```

### BillingQueryService — `:50052`

| Method | Request | Description |
|--------|---------|-------------|
| `GetInvoiceByOrder` | `order_id` | Returns invoice for a given order |
| `ListInvoices` | `status?, limit, offset` | Paginated invoice list with optional status filter |
| `GetBillingStats` | — | Total/paid/pending/cancelled counts + total revenue |

**Invoice state machine:**
```
PENDING ──► PAID
   │
   └──────► CANCELLED
```

## Running Locally

**Prerequisites:** Docker, Docker Compose

```bash
git clone https://github.com/Finechh/order-billing-system
cd order-billing-system
```

Create env files:

```bash
# order_service/.env
DB_HOST=postgres_order
DB_USER=order
DB_PASSWORD=orderpass
DB_NAME=order_db
DB_PORT=5432
KAFKA_BROKERS=kafka:9092
ORDER_CREATED_TOPIC=order-created-topic
ORDER_PAID_TOPIC=order-paid-topic
ORDER_CANCELLED_TOPIC=order-cancelled-topic
DB_SSLMODE=disable
OUTBOX_BATCH_SIZE=50
```

```bash
# billing_service/.env
DB_HOST=postgres_billing
DB_USER=billing
DB_PASSWORD=billingpass
DB_NAME=billing_db
DB_PORT=5432
KAFKA_BROKERS=kafka:9092
ORDER_CREATED_TOPIC=order-created-topic
ORDER_PAID_TOPIC=order-paid-topic
ORDER_CANCELLED_TOPIC=order-cancelled-topic
DB_SSLMODE=disable
```

```bash
docker-compose up --build
```

Services will be available at:
- `order_service` gRPC — `localhost:50051`
- `billing_service` gRPC — `localhost:50052`
- Prometheus — `http://localhost:9090`
- Grafana — `http://localhost:3000` (admin / admin)

> **Note:** On first startup you may see `Group Coordinator Not Available` errors in billing_service logs for ~15 seconds. This is expected — Kafka needs time to initialize the internal `__consumer_offsets` topic. The consumer retries automatically.

## Testing with grpcurl

## Testing with grpcurl

Both services have **gRPC reflection enabled** — you can explore the API without any `.proto` files:

```powershell
# List all available services
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext localhost:50052 list

# Describe a specific method
grpcurl -plaintext localhost:50051 describe order.v1.OrderService.CreateOrder
grpcurl -plaintext localhost:50052 describe billing.v1.BillingQueryService.GetBillingStats
```

> **Note for Windows (PowerShell):** use escaped double quotes `\"` instead of single quotes inside `-d`.

### Full end-to-end flow

**Step 1 — Create an order** (mixed currencies, auto-converted to USD)

```powershell
grpcurl -plaintext -d '{\"items\": [{\"product_id\": \"prod-1\", \"quantity\": 2, \"price\": {\"amount\": 1000, \"currency\": \"USD\"}}, {\"product_id\": \"prod-2\", \"quantity\": 1, \"price\": {\"amount\": 500, \"currency\": \"EUR\"}}]}' localhost:50051 order.v1.OrderService/CreateOrder
```

Response:
```json
{
  "order": {
    "id": "cf3b78c3-621e-4877-9eab-c22b4904e5cc",
    "items": [
      { "productId": "prod-1", "quantity": 2, "price": { "amount": "1000", "currency": "USD" } },
      { "productId": "prod-2", "quantity": 1, "price": { "amount": "500",  "currency": "EUR" } }
    ],
    "totalPrice": { "amount": "2545", "currency": "USD" },
    "status": "ORDER_STATUS_CREATED",
    "createdAt": "2026-02-25T11:03:16.313676906Z"
  }
}
```

> `prod-1`: 2 × 1000 USD = 2000 USD  
> `prod-2`: 1 × 500 EUR = 545 USD (rate 1.09)  
> **Total: 2545 USD** ✓

---

**Step 2 — Get order**

```powershell
grpcurl -plaintext -d '{\"order_id\": \"YOUR_ORDER_ID\"}' localhost:50051 order.v1.OrderService/GetOrder
```

---

**Step 3 — Check invoice in billing** (wait 2–3 seconds for outbox relay to publish to Kafka)

```powershell
grpcurl -plaintext -d '{\"order_id\": \"YOUR_ORDER_ID\"}' localhost:50052 billing.v1.BillingQueryService/GetInvoiceByOrder
```

Response:
```json
{
  "invoice": {
    "id": "5dabef80-031a-46e6-a5b4-7246a4f8b5b5",
    "orderId": "cf3b78c3-621e-4877-9eab-c22b4904e5cc",
    "amount": { "amount": "2545", "currency": "USD" },
    "status": "INVOICE_STATUS_PENDING",
    "createdAt": "2026-02-25T11:03:16.313676Z"
  }
}
```

---

**Step 4 — Mark order as paid**

```powershell
grpcurl -plaintext -d '{\"order_id\": \"YOUR_ORDER_ID\"}' localhost:50051 order.v1.OrderService/MarkOrderPaid
```

Response:
```json
{
  "order": {
    "id": "cf3b78c3-621e-4877-9eab-c22b4904e5cc",
    "totalPrice": { "amount": "2545", "currency": "USD" },
    "status": "ORDER_STATUS_PAID",
    "createdAt": "2026-02-25T11:03:16.313676Z"
  }
}
```

---

**Step 5 — Billing stats**

```powershell
grpcurl -plaintext -d '{}' localhost:50052 billing.v1.BillingQueryService/GetBillingStats
```

Response:
```json
{
  "totalInvoices": "5",
  "paidInvoices": "2",
  "pendingInvoices": "2",
  "cancelledInvoices": "1",
  "totalRevenue": { "amount": "5090", "currency": "USD" }
}
```

---

**Other commands**

```powershell
# Cancel an order
grpcurl -plaintext -d '{\"order_id\": \"YOUR_ORDER_ID\"}' localhost:50051 order.v1.OrderService/CancelOrder

# List invoices paginated, filter by status
grpcurl -plaintext -d '{\"status\": \"INVOICE_STATUS_PAID\", \"limit\": 10, \"offset\": 0}' localhost:50052 billing.v1.BillingQueryService/ListInvoices
```

## Running Tests

```bash
# order_service
cd order_service
go test ./internal/service/... -v -cover
# coverage: 63.9%

# billing_service
cd billing_service
go test ./internal/service/... -v -cover
# coverage: 88.5%
```

## Observability

Prometheus scrapes both services every 5 seconds. Available metrics:

| Metric | Description |
|--------|-------------|
| `*_grpc_requests_total` | gRPC request count by method and status code |
| `*_grpc_request_duration_seconds` | gRPC latency histogram |
| `*_kafka_messages_consumed_total` | Successfully processed Kafka messages |
| `*_kafka_messages_failed_total` | Failed messages sent to DLQ |
| `*_kafka_processing_duration_seconds` | Kafka handler latency |
| `*_db_open_connections` | PostgreSQL connection pool stats |
| `billing_business_invoices_created_total` | Business-level invoice counters |
| `order_business_orders_created_total` | Business-level order counters |

## Data Retention

This is a demo project. In production the following cleanup strategy would be applied:

- **`outbox_events`** — delete `SENT` records older than 7 days
- **`processed_events`** — delete records older than 30 days (deduplication window)
- **`orders` / `invoices`** — archive `PAID`/`CANCELLED` records older than 90 days to a cold storage table, then delete from hot table

## What's Not Implemented (known limitations)

- Single Kafka broker — no replication, not suitable for production
- Static currency rates — no live exchange rate feed
- No database migrations — uses GORM AutoMigrate
- No rate limiting on gRPC endpoints
- No distributed tracing (OpenTelemetry)