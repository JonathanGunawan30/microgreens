# Product Service

## Overview
Product Service is a microservice responsible for managing product catalogs, categories, and shopping carts. It provides a robust API for both administrative management and public product browsing, integrated with Elasticsearch for high-performance searching and RabbitMQ for event-driven updates.

## Architecture

This project follows **Hexagonal (Clean) Architecture**, ensuring a clear separation of concerns between business logic and infrastructure dependencies.

### Folder Structure
- `cmd/`: Entry points for the application (API server and workers).
- `config/`: Configuration management using Viper.
- `database/migrations/`: SQL migration files for database schema evolution.
- `internal/`:
    - `adapter/`: External integrations (the "infrastructure" layer).
        - `handler/`: HTTP request handlers (Echo).
        - `repository/`: Database implementations (GORM, Redis).
        - `message/`: Messaging implementations (RabbitMQ).
        - `storage/`: File storage implementations (Supabase).
    - `app/`: Application bootstrapping and dependency injection.
    - `core/`: Business logic (the "domain" layer).
        - `domain/`: Domain entities and models.
        - `service/`: Domain service implementations.
- `utils/`: Common utility functions.

## Event-Driven Architecture (EDA)

The service utilizes **RabbitMQ** for asynchronous communication and background processing:

### Published Events
- **Product Events**: When products are created, updated, or deleted, events are published to a `fanout` exchange (`EXCHANGE_PRODUCT_EVENT`). This allows other services (like Elasticsearch indexing) to react to product changes.

### Consumed Events
- **Elasticsearch Indexing**: A consumer listens for product events to sync the PostgreSQL database with Elasticsearch for fast searching.
- **Stock Updates**: Consumes events from other services (e.g., Order Service) to decrease product stock upon successful purchases.

### Workers
The application includes a dedicated worker command to run these consumers separately from the API server.

## Tech Stack
- **Language**: Go 1.24+
- **Web Framework**: Echo v4
- **Database**: PostgreSQL with GORM
- **Caching**: Redis
- **Search Engine**: Elasticsearch v9
- **Message Broker**: RabbitMQ
- **Object Storage**: Supabase Storage
- **Configuration**: Viper
- **Testing**: Testify, Mockery

## Prerequisites
- Go 1.24.6 or higher
- Docker & Docker Compose (for running dependencies)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI (for database migrations)

## Environment Variables
The application uses environment variables for configuration. See `.env.example` for a complete list of required variables and descriptions. Key categories include:
- `APP_*`: Application settings and JWT keys.
- `DATABASE_*`: PostgreSQL connection details.
- `REDIS_*`: Redis connection details.
- `RABBITMQ_*`: RabbitMQ connection and queue names.
- `ELASTICSEARCH_*`: Elasticsearch connection details.
- `SUPABASE_*`: Supabase storage credentials.

## Getting Started

### 1. Database Migrations
Run migrations up:
```bash
migrate -path database/migrations -database "postgresql://username:password@localhost:5432/product_db?sslmode=disable" up
```

Run migrations down:
```bash
migrate -path database/migrations -database "postgresql://username:password@localhost:5432/product_db?sslmode=disable" down
```

### 2. Running the Service
To run the API server:
```bash
go run main.go start
```

### 3. Running Workers
To run the background consumers (RabbitMQ):
```bash
go run main.go worker
```

## API Endpoints

### Public Endpoints
- `GET /products`: Get all products for the shop.
- `GET /products/featured`: Get featured products for the home page.
- `GET /products/:id`: Get public product details.
- `GET /categories`: Get all categories with children for the shop.
- `GET /categories/featured`: Get categories for the home page.

### Authenticated Endpoints (User)
- `GET /auth/cart`: Get current user's cart.
- `POST /auth/cart`: Add item to cart.
- `PATCH /auth/cart/decrease`: Decrease item quantity in cart.
- `DELETE /auth/cart/:id`: Remove specific item from cart.
- `DELETE /auth/cart`: Clear entire cart.
- `POST /auth/image-upload`: Upload image (for profiles/etc).

### Admin Endpoints
- `GET /admin/products`: List all products with advanced filtering.
- `POST /admin/products`: Create a new product.
- `GET /admin/products/:id`: Get product detail for admin.
- `PUT /admin/products/:id`: Update product.
- `DELETE /admin/products/:id`: Delete product.
- `POST /admin/categories`: Create category.
- `PUT /admin/categories/:id`: Update category.
- `DELETE /admin/categories/:id`: Delete category.
- `POST /admin/image-upload`: Upload product images.

## Testing
Run all unit tests:
```bash
go test ./tests/...
```

## Swagger Documentation
The API documentation is generated using Swag.

### Generate Docs
```bash
swag init
```

### Access Swagger UI
Once the server is running, access the UI at:
`http://localhost:8081/swagger/index.html` (port depends on your `APP_PORT` setting)
