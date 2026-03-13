# ALumiEye Backend API

A Go backend server for the ALumiEye MVP focused on email/password authentication.

## Features

- User registration with email + password
- User login with email + password
- JWT access token authentication
- Opaque refresh token with database-backed sessions
- **Swagger/OpenAPI documentation**
- Clean, extensible architecture ready for future features

## Tech Stack

- **Go** - Backend language
- **PostgreSQL** - Database
- **JWT** - Access tokens (short-lived, 15 minutes)
- **Argon2id** - Password hashing
- **Docker** - Containerization
- **Swagger** - API documentation

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Start all services
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

The API will be available at `http://localhost:8080`.

**Swagger UI**: `http://localhost:8080/docs/`

### Local Development

1. Start PostgreSQL:
```bash
docker run -d --name alumieye-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=alumieye \
  -p 5432:5432 \
  postgres:16-alpine
```

2. Run migrations:
```bash
psql -h localhost -U postgres -d alumieye -f migrations/001_initial_schema.up.sql
```

3. Configure secrets (optional for local dev - defaults in config.yml):
```bash
cp .env.example .env
# Edit .env with DATABASE_URL and JWT_SECRET if needed
```

4. Run the server:
```bash
make run
```

## API Documentation

### Swagger UI

Once the server is running, access the interactive API documentation at:

**http://localhost:8080/docs/**

The Swagger UI provides:
- Interactive API exploration
- Request/response examples
- Schema definitions
- Authentication testing with Bearer tokens

### Regenerating Swagger Docs

After modifying API annotations, regenerate the documentation:

```bash
# Install swag CLI (first time only)
make swagger-install

# Generate swagger docs
make swagger
```

## API Endpoints

### Auth Endpoints

#### Register
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "strong_password",
    "display_name": "Bruno"
  }'
```

Response:
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "display_name": "Bruno",
    "role": "user",
    "status": "active"
  },
  "tokens": {
    "access_token": "eyJ...",
    "refresh_token": "...",
    "expires_in": 900
  }
}
```

#### Login
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "strong_password"
  }'
```

#### Refresh Token
```bash
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "your_refresh_token_here"
  }'
```

#### Logout
```bash
curl -X POST http://localhost:8080/auth/logout \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "your_refresh_token_here"
  }'
```

#### Get Current User
```bash
curl -X GET http://localhost:8080/me \
  -H "Authorization: Bearer your_access_token_here"
```

Response:
```json
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "display_name": "Bruno",
    "role": "user",
    "status": "active"
  }
}
```

### Health Check
```bash
curl http://localhost:8080/health
```

## Project Structure

```
cmd/api/              # Application entry point
docs/                 # Generated Swagger documentation
internal/
  apierrors/          # API error handling
  auth/               # Authentication logic
    handlers.go       # HTTP handlers
    middleware.go     # JWT middleware
    service.go        # Business logic
    tokens.go         # JWT utilities
    types.go          # Request/response types
  config/             # Configuration management
  db/                 # Database connection
  http/               # HTTP utilities & router
  identity/           # User identity (multi-provider support)
  platform/
    crypto/           # Cryptographic utilities
  session/            # Session management
  user/               # User management
migrations/           # SQL migrations
```

## Configuration

**Public config** (safe to commit): `configs/config.yml`  
**Secrets** (env only): `.env` or environment variables

| Source | Variable | Description |
|--------|----------|-------------|
| config.yml | `app.env`, `app.port`, `app.service_name` | Environment, port, service name |
| config.yml | `server.access_token_ttl_minutes` | Access token lifetime (default: 15) |
| config.yml | `server.refresh_token_ttl_hours` | Refresh token lifetime (default: 720) |
| config.yml | `logging.level`, `logging.format` | Log level and format |
| env | `DATABASE_URL` | PostgreSQL connection string |
| env | `JWT_SECRET` | Secret key for JWT signing |

## Error Codes

| Code | Description |
|------|-------------|
| `validation_error` | Input validation failed |
| `email_already_exists` | Email is already registered |
| `invalid_credentials` | Wrong email or password |
| `unauthorized` | Missing or invalid token |
| `invalid_refresh_token` | Refresh token is invalid |
| `session_expired` | Session has expired |
| `user_blocked` | User account is blocked |
| `internal_error` | Internal server error |

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

## Make Commands

```bash
make help              # Show all available commands
make build             # Build the application
make run               # Run the application locally
make test              # Run tests
make swagger-install   # Install swag CLI tool
make swagger           # Generate Swagger documentation
make docker-up         # Start all services with Docker
make docker-down       # Stop all services
```

## Future Extensions

The codebase is structured to easily add:
- Google OAuth login
- Email verification
- Password reset
- Subscription plans
- Plan-based features
- Admin roles
