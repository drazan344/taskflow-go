# 🚀 TaskFlow API Testing Guide

Your TaskFlow application is successfully running! Here's how to test it:

## ✅ What's Working

1. **Database Setup**: PostgreSQL with all tables created
2. **Redis Connection**: Caching and sessions ready
3. **WebSocket Hub**: Real-time communication active
4. **HTTP Server**: Running on http://localhost:8080
5. **Health Check**: http://localhost:8080/health

## 🧪 API Testing Steps

### 1. Health Check
```bash
curl http://localhost:8080/health
```
Expected: `{"database":"up","redis":"up","status":"healthy",...}`

### 2. API Documentation
Visit: http://localhost:8080/docs/swagger/index.html

### 3. Test Registration (if user doesn't exist)
```bash
curl -X POST "http://localhost:8080/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@company.com",
    "password": "password123", 
    "first_name": "Jane",
    "last_name": "Smith",
    "tenant_name": "Test Company",
    "tenant_slug": "testco"
  }'
```

### 4. Test Login
```bash
curl -X POST "http://localhost:8080/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@company.com",
    "password": "password123"
  }'
```

### 5. Use the JWT Token
Save the `access_token` from login response, then:

```bash
export TOKEN="your_access_token_here"

# Test authenticated endpoint
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/auth/me"

# List tasks  
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/tasks"

# Create a task
curl -X POST "http://localhost:8080/api/v1/tasks" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My First Task",
    "description": "Testing the TaskFlow API",
    "priority": "high"
  }'
```

## 🌐 WebSocket Testing

1. Open `examples/websocket_client.html` in your browser
2. Get a JWT token from login above  
3. Paste the token and click "Connect"
4. You should see real-time connection established!

## 🔍 Architecture Highlights

**Multi-tenant Design**: Each request is isolated by `tenant_id`
**JWT Security**: Access + refresh token pattern
**Real-time Updates**: WebSocket hub with tenant-specific rooms
**Clean Architecture**: Models → Services → Handlers → HTTP

## 🎯 Next Steps for Learning

1. **Study the Code**: Follow the STUDY_GUIDE.md
2. **Add Features**: Try adding new endpoints
3. **Test Real-time**: Create tasks in one browser tab, watch updates in WebSocket client
4. **Explore Database**: Check PostgreSQL tables that were created
5. **Scale Up**: Add more services like email notifications

## 🐛 Troubleshooting

If you see errors:
1. Check database is running: `docker ps`
2. Check server logs in terminal
3. Verify JSON structure in API calls
4. Test health endpoint first

---

**Congratulations!** 🎉 You now have a production-ready, multi-tenant SaaS backend running locally.