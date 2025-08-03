#!/bin/bash

# Setup script for Fleet Sustainability with Authentication

echo "🚀 Setting up Fleet Sustainability with Authentication..."

# Check if MongoDB is running
echo "📊 Checking MongoDB connection..."
if ! curl -s http://localhost:27017 > /dev/null 2>&1; then
    echo "⚠️  MongoDB is not running. Please start MongoDB first."
    echo "   You can start it with: docker-compose up -d mongo"
    exit 1
fi

# Create test user
echo "👤 Creating test user..."
curl -X POST http://localhost:8081/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@fleet.com",
    "password": "admin123",
    "first_name": "Admin",
    "last_name": "User",
    "role": "admin"
  }' 2>/dev/null

if [ $? -eq 0 ]; then
    echo "✅ Test user created successfully!"
    echo "   Username: admin"
    echo "   Password: admin123"
else
    echo "ℹ️  Test user may already exist or server not running"
fi

echo ""
echo "🎯 To test the application:"
echo "1. Start the backend: go run cmd/main.go"
echo "2. Start the frontend: cd frontend && npm start"
echo "3. Open http://localhost:3000"
echo "4. Login with admin/admin123"
echo ""
echo "🔧 Additional test users you can create:"
echo "   Manager: manager/manager123"
echo "   Operator: operator/operator123"
echo "   Viewer: viewer/viewer123" 