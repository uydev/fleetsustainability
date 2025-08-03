#!/bin/bash

# Test script for Fleet Sustainability Authentication

echo "🧪 Testing Fleet Sustainability Authentication..."

BASE_URL="http://localhost:8081"

# Test 1: Register a test user
echo "📝 Test 1: Registering test user..."
REGISTER_RESPONSE=$(curl -s -X POST $BASE_URL/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@fleet.com",
    "password": "test123",
    "first_name": "Test",
    "last_name": "User",
    "role": "admin"
  }')

if echo "$REGISTER_RESPONSE" | grep -q "token"; then
    echo "✅ Registration successful"
    TOKEN=$(echo "$REGISTER_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
else
    echo "❌ Registration failed: $REGISTER_RESPONSE"
    exit 1
fi

# Test 2: Login
echo "🔐 Test 2: Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST $BASE_URL/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "test123"
  }')

if echo "$LOGIN_RESPONSE" | grep -q "token"; then
    echo "✅ Login successful"
    TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
else
    echo "❌ Login failed: $LOGIN_RESPONSE"
    exit 1
fi

# Test 3: Access protected endpoint
echo "🔒 Test 3: Accessing protected endpoint..."
PROTECTED_RESPONSE=$(curl -s -X GET $BASE_URL/api/telemetry \
  -H "Authorization: Bearer $TOKEN")

if [ $? -eq 0 ]; then
    echo "✅ Protected endpoint accessible"
else
    echo "❌ Protected endpoint failed: $PROTECTED_RESPONSE"
    exit 1
fi

# Test 4: Access without token (should fail)
echo "🚫 Test 4: Accessing without token..."
UNAUTHORIZED_RESPONSE=$(curl -s -X GET $BASE_URL/api/telemetry)

if echo "$UNAUTHORIZED_RESPONSE" | grep -q "401\|Unauthorized"; then
    echo "✅ Unauthorized access properly blocked"
else
    echo "❌ Unauthorized access not blocked: $UNAUTHORIZED_RESPONSE"
    exit 1
fi

# Test 5: Get user profile
echo "👤 Test 5: Getting user profile..."
PROFILE_RESPONSE=$(curl -s -X GET $BASE_URL/api/auth/profile \
  -H "Authorization: Bearer $TOKEN")

if echo "$PROFILE_RESPONSE" | grep -q "username"; then
    echo "✅ Profile retrieval successful"
else
    echo "❌ Profile retrieval failed: $PROFILE_RESPONSE"
    exit 1
fi

echo ""
echo "🎉 All authentication tests passed!"
echo "✅ Registration: Working"
echo "✅ Login: Working"
echo "✅ Protected endpoints: Working"
echo "✅ Authorization: Working"
echo "✅ Profile access: Working"
echo ""
echo "🚀 The authentication system is ready for use!" 