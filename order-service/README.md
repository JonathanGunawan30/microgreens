# Order Service

## Overview
Order Service is a microservice responsible for managing the entire lifecycle of customer orders in the Microservices Sayur ecosystem. It handles order creation, status tracking, and integrates with other services like Product and Payment through event-driven communication.

## Hexagonal (Clean) Architecture
This project follows Hexagonal Architecture (Ports and Adapters) to ensure separation of concerns and maintainability.

### Folder Structure
- `cmd/`: Entry points for the application (HTTP server and Workers).
- `config/`: Configuration logic and environment variable mapping.
- `database/`: Database migrations and schema definitions.
- `internal/`:
    - `core/`:
        - `domain/`: Business entities and models.
        - `service/`: Core business logic (Inbound Port implementation).
    - `adapter/`:
        - `handler/`: HTTP handlers (Inbound Adapters).
        - `repository/`: Database and Elasticsearch implementations (Outbound Adapters).
        - `message/`: RabbitMQ consumers and publishers (Outbound Adapters).
- `utils/`: Common utility functions, constants, and validators.
- `mocks/`: Auto-generated mocks for testing.

## Event-Driven Architecture
The service uses RabbitMQ for asynchronous communication and consistency across microservices.

### Published Events
- `OrderEvent`: Published when a new order is created.
- `ProductUpdateStock`: Published to trigger stock updates in the Product service.
- `PublisherUpdateStatus`: Published when an order status changes.
- `EmailUpdateStatus`: Published to trigger email notifications.

### Consumed Events
- `UserSnapshot`: Consumes user data changes to maintain a local snapshot.
- `ProductSnapshot`: Consumes product data changes to maintain a local snapshot.
- `PaymentEvent`: Consumes payment status updates to update order payment methods.

### Available Workers
- `worker-order`: Handles order indexing to Elasticsearch.
- `worker-payment`: Processes payment status updates.
- `worker-update-status`: Syncs status updates to Elasticsearch.
- `worker-user-snapshot`: Manages local user data snapshots.
- `worker-product-snapshot`: Manages local product data snapshots.

## Tech Stack
- **Language**: Go (1.25.0)
- **Framework**: Echo (v4)
- **ORM**: GORM (v1.25.10)
- **Database**: PostgreSQL
- **Caching/Session**: Redis
- **Search Engine**: Elasticsearch (v9)
- **Messaging**: RabbitMQ
- **Configuration**: Viper
- **CLI**: Cobra

## Prerequisites
- Go 1.25.0 or later
- Docker and Docker Compose
- PostgreSQL, Redis, RabbitMQ, Elasticsearch

## Environment Variables
See [.env.example](.env.example) for the full list of required environment variables.

## How to Run

### Database Migrations
Using `golang-migrate`:
```bash
# Run migrations up
migrate -path database/migrations -database "postgres://user:password@localhost:5432/order_db?sslmode=disable" up

# Run migrations down
migrate -path database/migrations -database "postgres://user:password@localhost:5432/order_db?sslmode=disable" down
```

### Running the Workers
```bash
# General worker command
go run main.go worker-order
go run main.go worker-payment
go run main.go worker-update-status
go run main.go worker-user-snapshot
go run main.go worker-product-snapshot
```

### Running the Service
```bash
go run main.go start
```

## API Endpoints

### Auth (Customer)
- `POST /auth/orders`: Create a new order (Distance check middleware applied)
- `GET /auth/orders`: Get all orders for the authenticated customer
- `GET /auth/orders/:id`: Get customer order details by ID
- `GET /auth/orders/:code/code`: Get order details by order code

### Admin
- `GET /admin/orders`: Get all orders (paginated)
- `GET /admin/orders/:id`: Get order details by ID
- `PATCH /admin/orders/:id/status`: Update order status

## Testing
To run all unit tests:
```bash
go test ./tests/...
```

## Swagger Documentation
### Generate Documentation
```bash
swag init
```
### Access Swagger UI
Once the service is running, access the documentation at:
`http://localhost:APP_PORT/swagger/index.html`
