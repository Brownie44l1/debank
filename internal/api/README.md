# API Layer

HTTP layer handling requests/responses only. No business logic here.

## Responsibilities
- Parse HTTP requests
- Validate input format
- Call appropriate service methods
- Format HTTP responses
- Handle HTTP errors

## Structure

### `/handlers`
HTTP endpoint handlers - one file per domain (wallet, auth, user)

### `/middleware`
- `auth.go` - JWT token verification
- `rate_limit.go` - Rate limiting per user/IP
- `cors.go` - CORS configuration
- `logger.go` - Request/response logging

### `/validators`
Input validation using Go validator tags and custom validators

### `router.go`
Central route registration and middleware setup

## Usage
```go
// In main.go
router := api.NewRouter(services, middlewares)
router.Run(":8080")