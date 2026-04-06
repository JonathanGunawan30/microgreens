# Notification Service

Notification Service is a microservice responsible for managing and delivering notifications to users through various channels like Email and Push (WebSockets). It follows Hexagonal (Clean) Architecture and is built with Go.

## Architecture

This project follows **Hexagonal (Clean) Architecture**, ensuring a clear separation of concerns between business logic and external infrastructure.

### Folder Structure
- `cmd/`: Entry point of the application and CLI commands (using Cobra).
- `config/`: Configuration management, including database, Redis, RabbitMQ, and environment variables.
- `database/migrations/`: Database schema migration files.
- `docs/`: Swagger documentation files.
- `internal/`: Main application code.
  - `adapter/`: External interfaces like HTTP handlers, RabbitMQ consumers, and database repositories.
    - `email/`: SMTP email sender implementation.
    - `handler/`: HTTP and WebSocket handlers.
    - `message/`: RabbitMQ consumers.
    - `repository/`: GORM-based database persistence.
    - `router/`: Route definitions for Echo.
  - `app/`: Application bootstrapping and server startup.
  - `core/`: Core business logic.
    - `domain/`: Business entities and models.
    - `port/`: Interface definitions for services and repositories.
    - `service/`: Domain service implementations.
- `tests/`: Unit tests and mocks.
- `utils/`: Common utility functions and constants.

## Event-Driven Architecture

The service uses **RabbitMQ** to handle asynchronous notification tasks. It consumes various events to trigger notifications:
- **Email Notifications**: Triggered for registration, password recovery, customer updates, and order status updates.
- **Push Notifications**: Real-time alerts delivered via WebSockets to users' dashboards.
- **Order Events**: Consumes order-related events from an exchange to notify admins about new orders.

### Available Consumers
- `NOTIF_EMAIL_VERIFICATION`: Sends verification emails.
- `NOTIF_EMAIL_FORGOT_PASSWORD`: Sends password reset links.
- `NOTIF_EMAIL_UPDATE_CUSTOMER`: Notifies about customer profile updates.
- `NOTIF_EMAIL_CREATE_CUSTOMER`: Notifies about new customer creation.
- `NOTIF_EMAIL_UPDATE_STATUS_ORDER`: Notifies about order status changes.
- `TypePush`: Handles general push notification events.
- `ORDER_EMAIL_QUEUE`: Admin notification for new orders via email.
- `ORDER_PUSH_QUEUE`: Admin notification for new orders via push.

## Real-time Notifications (WebSocket)

The service provides a WebSocket endpoint (`/ws`) for real-time notifications. When a notification is processed with the `push` type, it is broadcasted to the connected client based on their `user_id`.

## Tech Stack
- **Language**: Go (Golang)
- **Framework**: Echo (HTTP Server)
- **ORM**: GORM (PostgreSQL)
- **Message Broker**: RabbitMQ
- **Cache**: Redis
- **Documentation**: Swagger (swag)
- **Real-time**: Gorilla WebSocket
- **Validation**: go-playground/validator

## Prerequisites
- Go 1.25.0+
- PostgreSQL
- Redis
- RabbitMQ
- Docker (optional, for local development)

## Environment Variables

Copy `.env.example` to `.env` and fill in the values:

| Variable | Description |
|----------|-------------|
| `APP_PORT` | Port number for the application |
| `APP_ENV` | Application environment (development/production) |
| `SERVER_TIMEOUT` | Server timeout in seconds |
| `JWT_SECRET_KEY` | Secret key for JWT verification |
| `ADMIN_EMAIL` | Admin email for receiving order alerts |
| `ADMIN_ID` | Admin user ID for push notifications |
| `DATABASE_HOST` | PostgreSQL host |
| `DATABASE_PORT` | PostgreSQL port |
| `DATABASE_USER` | PostgreSQL username |
| `DATABASE_PASSWORD` | PostgreSQL password |
| `DATABASE_NAME` | PostgreSQL database name |
| `REDIS_HOST` | Redis host |
| `REDIS_PORT` | Redis port |
| `REDIS_PASSWORD` | Redis password |
| `REDIS_DB` | Redis database index |
| `RABBITMQ_HOST` | RabbitMQ host |
| `RABBITMQ_PORT` | RabbitMQ port |
| `RABBITMQ_USERNAME` | RabbitMQ username |
| `RABBITMQ_PASSWORD` | RabbitMQ password |
| `EMAIL_HOST` | SMTP server host |
| `EMAIL_PORT` | SMTP server port |
| `EMAIL_USERNAME` | SMTP username |
| `EMAIL_PASSWORD` | SMTP password |
| `EMAIL_FROM` | Sender email address |
| `EMAIL_TLS` | Use TLS for SMTP (true/false) |
| `ORDER_EVENT` | RabbitMQ exchange name for order events |

## Database Migrations

This project uses `golang-migrate`.

### Apply Migrations
```bash
migrate -path database/migrations -database "postgres://user:password@localhost:5432/dbname?sslmode=disable" up
```

### Rollback Migrations
```bash
migrate -path database/migrations -database "postgres://user:password@localhost:5432/dbname?sslmode=disable" down
```

## Running the Service

### Main Server & Consumers
The application runs both the HTTP server and all RabbitMQ consumers concurrently when started.

```bash
go run main.go start
```

## API Endpoints

### Auth Resource (Requires JWT Token)
- `GET /auth/notifications`: List all notifications for the authenticated user.
- `PATCH /auth/notifications/read-all`: Mark all notifications as read.
- `GET /auth/notifications/:id`: Get a specific notification by ID.
- `PATCH /auth/notifications/:id/read`: Mark a specific notification as read.

### WebSocket
- `GET /ws?user_id={id}`: Establish a WebSocket connection for real-time notifications.

### Health Check
- `GET /api/check`: Service health status.

## Unit Testing

To run unit tests:
```bash
go test ./tests/...
```

To run tests with coverage:
```bash
go test -cover ./tests/...
```

## Swagger Documentation

### Generate Docs
```bash
swag init -g internal/app/app.go -o docs
```

### Access UI
Once the server is running, you can access the Swagger UI at:
`http://localhost:{APP_PORT}/swagger/index.html`
