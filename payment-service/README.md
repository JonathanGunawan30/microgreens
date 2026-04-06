# Payment Service

## Overview
Payment Service is a microservice responsible for handling payment processing and webhooks. It serves as an intermediary between the order service, users, and the Midtrans payment gateway. The service processes cash-on-delivery (COD) and Midtrans digital payments, maintaining synchronization with user and order data through asynchronous events.

## Architecture
The project follows a **Hexagonal (Clean) Architecture**, emphasizing separation of concerns and testability:
- `cmd/`: Command-line entry points to start the server (`start`) or run background workers (`worker-order-snapshot`, `worker-user-snapshot`).
- `config/`: Application configuration structures mapped from environment variables.
- `database/migrations/`: SQL files for defining and updating the PostgreSQL database schema.
- `internal/`
  - `core/`: The heart of the application containing business logic without external dependencies.
    - `domain/entity/`: Database entities and Data Transfer Objects (DTOs).
    - `domain/model/`: Core business models.
    - `service/`: Business rules and application logic (e.g., `PaymentService`).
  - `adapter/`: External implementations (inbound and outbound).
    - `handler/`: Inbound HTTP handlers (Echo controllers).
    - `message/`: Inbound consumers and outbound publishers for RabbitMQ.
    - `repository/`: Outbound database repositories (GORM).
    - `Midtrans HTTP Client`: External API client.
  - `app/`: Wiring up dependencies and server initialization.
- `tests/`: Contains comprehensive unit tests (`tests/service/`, `tests/handler/`) and Mockery generated mocks (`tests/mocks/`).
- `utils/`: Reusable helpers for formatting, JWT validation, and input validation.

## Event-Driven Architecture
The service utilizes RabbitMQ to ensure high availability and data consistency without blocking synchronous requests.
- **Publishing Events:** After a successful or pending payment creation, the service publishes an update to the `EXCHANGE_PAYMENT_EVENT` exchange (`payment.event`).
- **Consuming Events:**
  - `worker-order-snapshot`: Consumes from `QUEUE_ORDER_SNAPSHOT_DB` on `EXCHANGE_ORDER_EVENT` to automatically upsert order details when orders are created or updated elsewhere in the system.
  - `worker-user-snapshot`: Consumes from `QUEUE_USER_SNAPSHOT_DB` on `EXCHANGE_USER_EVENT` to maintain a localized, up-to-date snapshot of users.

## Tech Stack
- **Language**: Go (v1.25.0)
- **Web Framework**: Echo (v4.15.1)
- **Database ORM**: GORM with PostgreSQL driver
- **Message Broker**: RabbitMQ (`amqp091-go`)
- **Cache / Session**: Redis (`go-redis/v9`)
- **Payment Gateway**: Midtrans (`midtrans-go`)
- **Testing**: `testify`, `mockery`, built-in `httptest`
- **Validation**: `go-playground/validator/v10`
- **CLI Framework**: Cobra

## Prerequisites
- **Go**: 1.25.0 or newer
- **PostgreSQL**: Running instance
- **Redis**: Running instance
- **RabbitMQ**: Running instance
- **Golang-Migrate**: For executing database migrations (https://github.com/golang-migrate/migrate)
- **Swag**: For generating Swagger API documentation (`go install github.com/swaggo/swag/cmd/swag@latest`)

## Environment Variables
See the `.env.example` file for a list of required variables and their descriptions. Ensure these are defined in a `.env` file in the root of the service before starting.

## Database Migrations
To set up the initial schema, run the golang-migrate CLI from the project root. Make sure to replace the placeholder connection string with your actual local configuration.

```bash
migrate -path database/migrations -database "postgresql://postgres:postgres@localhost:5433/sayur_payment_service?sslmode=disable" up
```

*(Note: To revert the latest migration, use `down 1` instead of `up`)*

## Available Workers / Consumers
Workers process background events independently of the main API server. Run them in separate terminal instances:

1. **Order Snapshot Worker:** Synchronizes order data from other services.
   ```bash
   go run main.go worker-order-snapshot
   ```

2. **User Snapshot Worker:** Synchronizes user data from other services.
   ```bash
   go run main.go worker-user-snapshot
   ```

## Running the Service
To start the main Echo HTTP API server:

```bash
go run main.go start
```
The server will boot up and bind to the `APP_PORT` specified in your `.env`.

## Available API Endpoints

### Public Endpoints
- `POST /payments/webhook`: Midtrans webhook callback to process transaction status updates asynchronously.

### Authenticated Endpoints (Customer / General User)
*(Requires `Authorization: Bearer <token>`)*
- `POST /auth/payments`: Create a new payment (COD or Midtrans).
- `GET /auth/payments`: List payments belonging to the authenticated customer.
- `GET /auth/payments/:id`: Get specific details of a payment belonging to the authenticated customer.

### Admin Endpoints (Super Admin)
*(Requires `Authorization: Bearer <token>` and Admin Role)*
- `GET /admin/payments`: List all payments across all users.
- `GET /admin/payments/:id`: Retrieve details for any specific payment in the system.

## Running Unit Tests
The project relies on a comprehensive suite of unit tests, especially for the Handler and Service layers. Run the entire suite using:

```bash
go test -v ./tests/...
```

## Swagger Documentation
The project includes Swagger annotations on the HTTP handler methods.
To generate or update the documentation:

```bash
swag init -g internal/adapter/handler/payment_handler.go
```
*(You may need to start the service and navigate to `/swagger/index.html` depending on if the `echo-swagger` middleware is mounted in your `app.go` setup).*
