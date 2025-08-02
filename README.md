# Fleet Sustainability Dashboard (IoT Simulation)

This project is a Go backend service for the Fleet Sustainability Dashboard, designed to simulate IoT data and provide analytics for fleet management and sustainability tracking.

## Project Structure
- `cmd/` — Main application entry points
- `internal/` — Private application and library code
- `pkg/` — Public libraries for use by other projects
- `api/` — API definitions and documentation
- `scripts/` — Helper scripts for development and operations
- `build/` — Packaging and CI/CD configurations
- `configs/` — Configuration files
- `test/` — External tests and test data

## Features
- IoT data simulation for fleet vehicles
- RESTful API for data access and analytics
- Modular, idiomatic Go codebase
- Security best practices (input validation, JWT auth, HTTPS)
- Ready for CI/CD and containerized deployment

## Stateless, Horizontally Scalable Design
This backend is designed to be stateless and horizontally scalable:
- All state is stored in external systems (MongoDB, etc.), not in memory.
- No session or user state is kept in the application process.
- Multiple instances can be run behind a load balancer for high availability and scale-out.
- Configuration is via environment variables or .env files, supporting container orchestration.

## API Authentication
All main API endpoints require a JWT Bearer token in the `Authorization` header. Set the `JWT_SECRET` environment variable to configure the secret.

## HTTPS Support
Set `USE_HTTPS=true` and provide `TLS_CERT_FILE` and `TLS_KEY_FILE` environment variables to enable HTTPS in production.

## Quick API Usage Example
```
curl -H "Authorization: Bearer <your-jwt>" https://localhost:8080/api/telemetry
```

## Getting Started
1. Clone the repository
2. Run `go mod tidy` to install dependencies
3. Build and run the main application in `cmd/`

## License
MIT 