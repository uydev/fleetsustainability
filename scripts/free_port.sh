#!/bin/bash

# Free Ports 8080, 8081, 8082, and 3000 Script

echo "🔧 Freeing up ports 8080, 8081, 8082, and 3000..."

# Function to free a port
free_port() {
    local port=$1
    echo ""
    echo "📡 Checking port $port..."
    
    # Check if port is in use
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null ; then
        echo "⚠️  Port $port is in use by:"
        lsof -i :$port
        
        echo ""
        echo "🔄 Killing processes on port $port..."
        
        # Kill processes on port
        lsof -ti:$port | xargs kill -9 2>/dev/null || {
            echo "⚠️  Could not kill processes without sudo, trying with sudo..."
            sudo lsof -ti:$port | xargs kill -9 2>/dev/null || {
                echo "❌ Could not kill processes on port $port"
                echo "💡 Try running manually: sudo lsof -ti:$port | xargs kill -9"
            }
        }
        
        # Wait a moment for port to be freed
        sleep 2
        
        # Check if port is now free
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null ; then
            echo "❌ Port $port is still in use"
        else
            echo "✅ Port $port is now free"
        fi
    else
        echo "✅ Port $port is already free"
    fi
}

# Free port 8080 (backend internal)
free_port 8080

# Free port 8081 (backend external)
free_port 8081

# Free port 8082 (Mongo Express)
free_port 8082

# Free port 3000 (frontend)
free_port 3000

echo ""
echo "🎉 Port freeing complete!" 