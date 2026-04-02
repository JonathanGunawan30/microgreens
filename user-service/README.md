# User Service (MicroGreens)

## Project Overview
The `user-service` is a core microservice responsible for user management, authentication, and authorization within the broader MicroGreens application ecosystem. It handles user registration, secure authentication (JWT), profile management, role assignments, and publishes critical domain events related to user state changes to a message broker for other microservices to consume.

## Hexagonal (Clean) Architecture
This project rigidly adheres to the Hexagonal Architecture (also known as Ports and Adapters) to ensure separation of concerns, testability, and independence from external frameworks and databases.

The folder structure reflects this paradigm:

- **`cmd/`**: The entry point of the application. Contains the Cobra CLI commands (`root.go`, `start.go`) to bootstrap the service.
- **`config/`**: Configuration loaders (`viper`), environment variable bindings, and client initializations for Redis, RabbitMQ, and PostgreSQL.
- **`database/`**: Contains raw SQL migration files (`migrations/`) and seed data (`seeds/`) to initialize the database schema and default roles/admins.
- **`internal/`**: The heart of the application, divided into Core and Adapters.
  - **`internal/core/`**: The business logic layer.
    - **`domain/`**: Contains pure Go structs (`entity`, `model`) representing the business objects (e.g., `UserEntity`, `RoleEntity`) with absolutely no external dependencies.
    - **`service/`**: Contains the business logic implementations (`UserService`, `RoleService`, `JwtService`). These dictate "what" the application does and interact with external systems exclusively through interfaces.
  - **`internal/adapter/`**: The infrastructure layer. Implements the interfaces defined in the core.
    - **`handler/`**: HTTP delivery mechanisms (`echo` framework controllers) to process incoming web requests and shape responses. Contains `request` and `response` payload definitions.
    - **`repository/`**: Database interactions (using `gorm` and PostgreSQL) mapping domain entities to database models.
    - **`message/`**: Message broker publishers (RabbitMQ) for dispatching events out of the system.
    - **`storage/`**: External storage implementations (Supabase) for profile image uploads.
  - **`internal/app/`**: Application bootstrapping, wiring up the dependencies (repositories to services, services to handlers), and starting the Echo server.
- **`tests/`**: Comprehensive unit testing suites for both the `service` layer and the HTTP `handler` layer using `testify` and generated `mockery` mocks.

## Event-Driven Architecture
The `user-service` is a fundamental producer in the event-driven ecosystem. It utilizes **RabbitMQ** to asynchronously broadcast state changes, preventing synchronous coupling with other services.

- **Publisher**: When key actions occur (e.g., a user successfully verifies their account), the service publishes an event (e.g., `UserEvent`) to a designated exchange (`user.event`).
- **Events Published**: 
  - User verification/registration events containing payload data like `UserID`, `Name`, `Email`, `Phone`, and `Address`.
- **Workers**: *Note: This specific service acts primarily as a producer for user events and handles HTTP requests; it currently does not run dedicated background workers for consuming events from other services. Other downstream microservices consume the `user.event` exchange.*

## Tech Stack
Based on the `go.mod`, the project utilizes the following modern Go stack:
- **Go**: v1.24+
- **Web Framework**: [Echo v4](https://echo.labstack.com/)
- **Database ORM**: [GORM](https://gorm.io/) with PostgreSQL driver
- **Message Broker**: [RabbitMQ AMQP091](https://github.com/rabbitmq/amqp091-go)
- **Caching & Sessions**: [Redis](https://github.com/redis/go-redis/v9)
- **Authentication**: JWT (`github.com/golang-jwt/jwt/v5`) & bcrypt hashing
- **Validation**: `go-playground/validator/v10`
- **Configuration**: Viper (`spf13/viper`)
- **CLI**: Cobra (`spf13/cobra`)
- **Cloud Storage**: Supabase Storage (`supabase-community/storage-go`)
- **API Documentation**: Swaggo (`swaggo/swag` & `swaggo/echo-swagger`)
- **Testing**: Testify (`stretchr/testify`) and Mockery

## Prerequisites
- **Go**: 1.24 or newer
- **PostgreSQL**: Running instance (v13+)
- **Redis**: Running instance
- **RabbitMQ**: Running instance
- **golang-migrate**: Installed for running database migrations

## Environment Variables
The application is configured heavily through environment variables. See the `.env.example` file for a complete list of required configurations and their descriptions. 
You must copy `.env.example` to `.env` and fill in your actual credentials.

## Database Migrations
This project uses [golang-migrate/migrate](https://github.com/golang-migrate/migrate) to manage database schema changes.

Ensure your `.env` is configured correctly, then run the migrations against your Postgres database:

**Up (Apply migrations):**
```bash
migrate -path database/migrations -database "postgresql://postgres:postgres@localhost:5433/sayur_user_service?sslmode=disable" -verbose up
```

**Down (Rollback migrations):**
```bash
migrate -path database/migrations -database "postgresql://postgres:postgres@localhost:5433/sayur_user_service?sslmode=disable" -verbose down
```

## Running the Service
To run the service locally for development:

1. **Ensure all infrastructure (Postgres, Redis, RabbitMQ) is running.**
2. **Apply migrations.**
3. **Start the service:**
```bash
go run main.go start
```
*Alternatively, you can use a live-reloading tool like [Air](https://github.com/cosmtrek/air) if configured in your environment.*

## Available API Endpoints

### Authentication (Public)
- `POST /signin`: Authenticate user and receive a JWT.
- `POST /signup`: Register a new user account.
- `POST /forgot-password`: Request a password reset link to be sent via email.
- `GET /verify-account`: Verify a user account using a token.
- `PUT /update-password`: Update a password using a reset token.

### Protected User Endpoints (Requires Bearer JWT)
- `POST /auth/logout`: Revoke the current user's session (clears Redis).
- `GET /auth/profile`: Get the authenticated user's profile data.
- `PUT /auth/profile`: Update the authenticated user's profile data.
- `PATCH /auth/profile/password`: Change the authenticated user's password.
- `POST /auth/profile/image-upload`: Upload a new profile image (multipart/form-data).

### Admin Endpoints (Requires Admin Bearer JWT)
**Customers Management:**
- `GET /admin/customers`: List paginated customers.
- `POST /admin/customers`: Create a new customer.
- `GET /admin/customers/:id`: Get customer by ID.
- `PUT /admin/customers/:id`: Update a customer.
- `DELETE /admin/customers/:id`: Delete a customer.

**Roles Management:**
- `GET /admin/roles`: List all roles.
- `POST /admin/roles`: Create a new role.
- `GET /admin/roles/:id`: Get role by ID.
- `PUT /admin/roles/:id`: Update a role.
- `DELETE /admin/roles/:id`: Delete a role.

## Running Unit Tests
The project features comprehensive unit tests for both the `service` and `handler` layers, utilizing mocked interfaces to isolate logic.

Run the entire test suite:
```bash
go test -v ./tests/...
```

To run specific layer tests:
```bash
go test -v ./tests/service/...
go test -v ./tests/handler/...
```

## API Documentation (Swagger)
Swagger documentation is automatically generated using `swaggo`.

1. **Generate/Update Docs:** Whenever you modify the Swaggo annotations in the handlers or the `main`/`app` setup, regenerate the documentation:
   ```bash
   swag init -g internal/app/app.go --parseDependency --parseInternal
   ```
2. **Access Swagger UI:** Run the application (`go run main.go start`), then navigate your browser to:
   ```
   http://localhost:8080/swagger/index.html
   ```
