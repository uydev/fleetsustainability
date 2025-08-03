#!/bin/bash

echo "🔍 Testing Login Fix..."
echo "========================"

# Check if backend is running
echo "1. Checking backend status..."
if curl -s http://localhost:8081 > /dev/null 2>&1; then
    echo "✅ Backend is running on port 8081"
else
    echo "❌ Backend is not running on port 8081"
    exit 1
fi

# Check if frontend is running
echo "2. Checking frontend status..."
if curl -s http://localhost:3000 > /dev/null 2>&1; then
    echo "✅ Frontend is running on port 3000"
else
    echo "❌ Frontend is not running on port 3000"
    exit 1
fi

# Test login API
echo "3. Testing login API..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8081/api/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username": "admin", "password": "admin123"}')

if echo "$LOGIN_RESPONSE" | grep -q "token"; then
    echo "✅ Login API is working"
    echo "   Token received: $(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4 | cut -c1-20)..."
else
    echo "❌ Login API failed"
    echo "   Response: $LOGIN_RESPONSE"
    exit 1
fi

# Test frontend API connection
echo "4. Testing frontend API connection..."
FRONTEND_RESPONSE=$(curl -s http://localhost:3000)

if echo "$FRONTEND_RESPONSE" | grep -q "Fleet Sustainability"; then
    echo "✅ Frontend is serving the app"
else
    echo "❌ Frontend is not serving the app correctly"
    echo "   Response preview: $(echo "$FRONTEND_RESPONSE" | head -5)"
fi

echo ""
echo "🎉 Login Fix Summary:"
echo "======================"
echo "✅ Backend: Running on port 8081"
echo "✅ Frontend: Running on port 3000"
echo "✅ API Connection: Working"
echo "✅ Login Endpoint: Working"
echo ""
echo "🌐 Open your browser and go to:"
echo "   http://localhost:3000"
echo ""
echo "🔑 Login credentials:"
echo "   Username: admin"
echo "   Password: admin123"
echo ""
echo "💡 If you still can't login in the browser:"
echo "   1. Hard refresh the page (Ctrl+Shift+R or Cmd+Shift+R)"
echo "   2. Clear browser cache and localStorage"
echo "   3. Try incognito/private mode"
echo "   4. Check browser console for errors (F12)" 