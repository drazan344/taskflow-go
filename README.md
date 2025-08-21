# TaskFlow - Multi-Tenant Task Management Platform

A production-ready, multi-tenant task management SaaS platform built with Go, showcasing modern backend development practices.

## Features

- 🏢 **Multi-tenant Architecture** - Complete tenant isolation and management
- 🔐 **JWT Authentication** - Secure authentication with refresh tokens
- 👥 **Role-based Access Control** - Admin, Manager, and User roles
- ⚡ **Real-time Updates** - WebSocket-based live collaboration
- 📧 **Email Notifications** - Task assignments and due date reminders
- 🔄 **Background Processing** - Async job processing with Redis
- 📊 **Analytics Dashboard** - Team productivity insights
- 🔍 **GraphQL API** - Flexible query interface
- 🛡️ **Rate Limiting** - Per-tenant usage control
- 📝 **Comprehensive API** - RESTful endpoints with OpenAPI docs

## Technology Stack

- **Framework**: Gin HTTP framework
- **Database**: PostgreSQL with GORM
- **Cache**: Redis for sessions and caching
- **Authentication**: JWT with refresh tokens
- **Real-time**: WebSocket with Gorilla
- **Background Jobs**: Asynq (Redis-based)
- **Testing**: Testify framework
- **Documentation**: Swagger/OpenAPI
- **Containerization**: Docker

## Project Structure

```
taskflow-go/
├── cmd/
│   ├── api/           # Main API server
│   └── worker/        # Background job worker
├── internal/
│   ├── config/        # Configuration management
│   ├── database/      # Database connection and setup
│   ├── models/        # Data models
│   ├── handlers/      # HTTP handlers
│   ├── middleware/    # HTTP middleware
│   ├── services/      # Business logic
│   ├── repositories/  # Data access layer
│   ├── auth/          # Authentication logic
│   ├── websocket/     # WebSocket handlers
│   └── jobs/          # Background job handlers
├── pkg/
│   ├── logger/        # Structured logging
│   ├── utils/         # Utility functions
│   └── errors/        # Error handling
├── migrations/        # Database migrations
├── docs/              # API documentation
├── scripts/           # Build and deployment scripts
├── deployments/       # Docker and k8s configs
└── test/              # Integration tests
```

## Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+

### Installation

1. Clone the repository:
```bash
git clone https://github.com/kayal/taskflow-go.git
cd taskflow-go
```

2. Install dependencies:
```bash
go mod download
```

3. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Run database migrations:
```bash
go run cmd/migrate/main.go up
```

5. Start the server:
```bash
go run cmd/api/main.go
```

## API Documentation

Once the server is running, visit:
- Swagger UI: http://localhost:8080/docs/swagger/index.html
- GraphQL Playground: http://localhost:8080/graphql

## Testing

Run the test suite:
```bash
go test -v ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.