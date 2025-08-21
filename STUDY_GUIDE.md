# TaskFlow Study Guide 📚

## How to Study This Project

This guide helps you understand the codebase progressively, from basic concepts to advanced patterns.

## 🎯 Phase 1: Foundation (Week 1-2)

### Day 1-2: Project Structure
**Files to study first:**
```
📁 TaskFlow/
├── README.md                    ← Start here
├── go.mod                       ← Dependencies overview
├── cmd/api/main.go             ← Application entry point
├── internal/config/config.go   ← Configuration management
└── .env.example                ← Environment setup
```

**Learning Goals:**
- [ ] Understand Go modules (`go mod`)
- [ ] Learn project layout conventions
- [ ] Grasp dependency injection concepts

**Exercises:**
1. Run `go mod graph` to see dependency tree
2. Trace how config flows from `.env` → `config.go` → `main.go`
3. Identify all the services initialized in `main.go`

### Day 3-4: Data Models
**Files to study:**
```
📁 internal/models/
├── base.go        ← Common patterns for all models
├── tenant.go      ← Multi-tenancy core concept
├── user.go        ← Authentication and roles
└── task.go        ← Main business logic
```

**Learning Goals:**
- [ ] Understand GORM associations and relationships
- [ ] Learn about embedded structs (`BaseModel`, `TenantModel`)
- [ ] Grasp multi-tenant data isolation patterns

**Exercises:**
1. Draw the database schema on paper
2. Identify all foreign key relationships
3. Understand how `tenant_id` provides isolation

### Day 5-7: Database Layer
**Files to study:**
```
📁 internal/database/
├── database.go    ← PostgreSQL connection and GORM setup
└── redis.go       ← Redis caching and sessions
```

**Learning Goals:**
- [ ] Database connection management
- [ ] Connection pooling concepts
- [ ] Caching strategies with Redis

## 🚀 Phase 2: Core Services (Week 3-4)

### Day 8-10: Authentication System
**Study order:**
```
1. internal/auth/jwt.go          ← JWT token generation/validation
2. internal/auth/service.go      ← Business logic (login, register)
3. internal/middleware/auth.go   ← Request authentication
```

**Learning Goals:**
- [ ] JWT token lifecycle (access + refresh)
- [ ] Password hashing with bcrypt
- [ ] Session management
- [ ] Security best practices

**Exercises:**
1. Trace a login request from handler → service → JWT creation
2. Understand how middleware validates tokens
3. Test JWT generation manually with online tools

### Day 11-14: HTTP Layer
**Study order:**
```
1. internal/handlers/auth.go     ← Authentication endpoints
2. internal/handlers/user.go     ← User management
3. internal/handlers/task.go     ← Core business logic
4. internal/middleware/          ← Cross-cutting concerns
```

**Learning Goals:**
- [ ] RESTful API design patterns
- [ ] Request validation and error handling
- [ ] Middleware chain execution
- [ ] Response formatting standards

## ⚡ Phase 3: Advanced Features (Week 5-6)

### Day 15-18: Real-time Communication
**Study order:**
```
1. internal/websocket/hub.go     ← WebSocket connection manager
2. internal/websocket/client.go  ← Individual client handling
3. internal/handlers/websocket.go ← WebSocket API endpoints
```

**Learning Goals:**
- [ ] WebSocket protocol basics
- [ ] Concurrent programming with goroutines
- [ ] Hub pattern for managing connections
- [ ] Real-time message broadcasting

**Exercises:**
1. Open `examples/websocket_client.html` in browser
2. Connect to WebSocket and send messages
3. Understand the hub-and-spoke pattern

### Day 19-21: Background Processing
**Study order:**
```
1. internal/jobs/types.go        ← Job definitions and payloads
2. Look at Asynq documentation   ← Understanding the job queue
```

**Learning Goals:**
- [ ] Asynchronous job processing
- [ ] Queue-based architecture
- [ ] Job retry and failure handling
- [ ] Distributed task processing

## 🎨 Phase 4: Production Patterns (Week 7-8)

### Day 22-25: Middleware Deep Dive
**Study each middleware:**
```
📁 internal/middleware/
├── auth.go         ← Authentication and authorization
├── cors.go         ← Cross-origin resource sharing
├── error.go        ← Centralized error handling
├── logging.go      ← Request/response logging
└── rate_limit.go   ← API rate limiting
```

**Learning Goals:**
- [ ] Middleware composition patterns
- [ ] Error handling strategies
- [ ] Observability (logging, metrics)
- [ ] Security considerations (CORS, rate limiting)

### Day 26-28: Configuration & Deployment
**Study:**
```
📁 Files:
├── Dockerfile              ← Containerization
├── docker-compose.yml      ← Multi-service orchestration
├── .env.example           ← Configuration management
└── Makefile               ← Build automation
```

**Learning Goals:**
- [ ] Container best practices
- [ ] Multi-stage Docker builds
- [ ] Service orchestration
- [ ] Production deployment strategies

## 🧪 Phase 5: Testing & Quality (Week 9-10)

### Study Testing Patterns
```
📁 test/                    ← Integration tests (to be created)
├── handlers_test.go
├── auth_test.go
└── integration_test.go
```

**Learning Goals:**
- [ ] Unit testing with testify
- [ ] HTTP endpoint testing
- [ ] Database testing patterns
- [ ] Mocking external dependencies

## 💡 Key Concepts to Master

### 1. Multi-Tenancy Patterns
```go
// Shared Database, Shared Schema approach
type TenantModel struct {
    BaseModel
    TenantID uuid.UUID `gorm:"type:uuid;not null;index"`
}

// Every query must include tenant isolation
db.Where("tenant_id = ?", tenantID).Find(&tasks)
```

### 2. Middleware Chain Pattern
```go
// Each middleware wraps the next one
router.Use(middleware.RequestIDMiddleware())     // 1st
router.Use(middleware.LoggerMiddleware(logger))  // 2nd
router.Use(middleware.AuthMiddleware(...))       // 3rd
```

### 3. Dependency Injection
```go
// Services depend on interfaces, not concrete types
type UserService struct {
    db     *gorm.DB      // Database dependency
    logger *logger.Logger // Logging dependency
}

// Inject dependencies at startup
userService := NewUserService(db, logger)
```

### 4. Error Handling Strategy
```go
// Custom error types with context
type AppError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Err     error  `json:"-"`
}

// Centralized error handling in middleware
func ErrorHandler(log *logger.Logger) gin.HandlerFunc {
    // Handle all errors consistently
}
```

### 5. Real-time Communication
```go
// Hub pattern for WebSocket management
type Hub struct {
    tenants    map[uuid.UUID]*TenantRoom
    register   chan *Client
    unregister chan *Client
    broadcast  chan *Message
}
```

## 🔧 Hands-on Exercises

### Exercise 1: Add a New Feature
Try adding a "Project Comments" feature:
1. Create the model in `internal/models/`
2. Add handlers in `internal/handlers/`
3. Create API endpoints
4. Test with Postman or curl

### Exercise 2: Understanding the Flow
Trace a complete request:
```
HTTP Request → Middleware Chain → Handler → Service → Repository → Database
     ↓
HTTP Response ← Error Handler ← Business Logic ← Data Layer
```

### Exercise 3: WebSocket Testing
1. Start the server: `go run cmd/api/main.go`
2. Open `examples/websocket_client.html`
3. Connect and send messages
4. Watch the server logs

## 📖 Additional Resources

### Go-Specific Learning:
- **Effective Go**: https://golang.org/doc/effective_go.html
- **Go by Example**: https://gobyexample.com/
- **Go Concurrency Patterns**: Study goroutines and channels

### Architecture Patterns:
- **Clean Architecture**: Uncle Bob's architecture principles
- **Domain-Driven Design**: Business logic organization
- **Microservices Patterns**: Service decomposition strategies

### SaaS Development:
- **Multi-tenancy Patterns**: Database isolation strategies
- **API Design**: RESTful principles and versioning
- **Authentication**: JWT, OAuth2, session management

## 🎓 Mastery Checklist

By the end of your study, you should be able to:

- [ ] **Explain the multi-tenant architecture** and why we chose this approach
- [ ] **Trace a request** from HTTP to database and back
- [ ] **Understand all middleware** and their purposes
- [ ] **Implement JWT authentication** from scratch
- [ ] **Design WebSocket communication** patterns
- [ ] **Structure a Go project** following best practices
- [ ] **Write production-ready code** with proper error handling
- [ ] **Deploy the application** using Docker

## 🚨 Common Pitfalls to Avoid

1. **Don't skip the foundation** - Understand Go basics first
2. **Don't memorize** - Understand the "why" behind patterns
3. **Practice coding** - Don't just read, implement features
4. **Test your understanding** - Build something similar
5. **Study real-world examples** - Look at other open-source projects

---

Remember: This is a **reference implementation** showing production-ready patterns. Take time to understand each concept before moving to the next phase.

Happy learning! 🚀