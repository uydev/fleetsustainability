#!/bin/bash

# Start Backend Script - Automatically finds available port

echo "🚀 Starting Fleet Sustainability Backend..."

# Function to check if port is available
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null ; then
        return 1  # Port is in use
    else
        return 0  # Port is available
    fi
}

# Find available port starting from 8080
PORT=8080
while ! check_port $PORT; do
    echo "⚠️  Port $PORT is in use, trying next port..."
    PORT=$((PORT + 1))
    if [ $PORT -gt 8090 ]; then
        echo "❌ Could not find available port between 8080-8090"
        exit 1
    fi
done

echo "✅ Using port $PORT"

# Set environment variables
export PORT=$PORT
export MONGO_URI=mongodb://root:example@localhost:27017
export MONGO_DB=fleet
export JWT_SECRET=your-secret-key-change-in-production

echo "📊 Backend will be available at: http://localhost:$PORT"
echo "🔐 Authentication endpoints:"
echo "   - Login: http://localhost:$PORT/api/auth/login"
echo "   - Register: http://localhost:$PORT/api/auth/register"
echo ""
echo "💡 To stop the server, press Ctrl+C"
echo ""

# Start the backend
go run cmd/main.go 