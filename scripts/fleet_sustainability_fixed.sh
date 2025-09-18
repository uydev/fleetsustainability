#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if port is in use
check_port() {
    if lsof -i :$1 >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to kill process on port
kill_port() {
    local port=$1
    local pid=$(lsof -ti:$port)
    if [ ! -z "$pid" ]; then
        # Check if it's a Node.js process before killing
        local process_name=$(ps -p $pid -o comm= 2>/dev/null)
        if [[ "$process_name" == *"node"* ]] || [[ "$process_name" == *"npm"* ]]; then
            print_warning "Killing Node.js process on port $port (PID: $pid)"
            kill -9 $pid 2>/dev/null
            sleep 2
        else
            print_warning "Port $port is in use by $process_name (PID: $pid) - not killing non-Node.js process"
        fi
    fi
}

# Function to free a specific port - IMPROVED VERSION
free_port() {
    local port=$1
    print_status "Checking port $port..."
    
    # Check if port is in use
    if check_port $port; then
        print_warning "Port $port is in use by:"
        lsof -i :$port
        
        # Get all PIDs using the port
        local pids=$(lsof -ti:$port)
        local killed_any=false
        
        for pid in $pids; do
            local process_name=$(ps -p $pid -o comm= 2>/dev/null)
            local full_command=$(ps -p $pid -o command= 2>/dev/null)
            
            # Skip system processes that we shouldn't kill
            if [[ "$process_name" == "kernel" ]] || [[ "$process_name" == "system" ]] || [[ "$full_command" == *"kernel"* ]]; then
                print_warning "Skipping system process $process_name (PID: $pid)"
                continue
            fi
            
            # Kill the process
            print_status "Killing process $process_name on port $port (PID: $pid)"
            kill -9 $pid 2>/dev/null
            killed_any=true
        done
        
        if [ "$killed_any" = true ]; then
            # Wait a moment for port to be freed
            sleep 3
            
            # Check if port is now free
            if check_port $port; then
                print_error "Port $port is still in use after killing processes"
                return 1
            else
                print_status "Port $port is now free"
                return 0
            fi
        else
            print_warning "No processes were killed on port $port"
            return 1
        fi
    else
        print_status "Port $port is already free"
        return 0
    fi
}

# Test the function
echo "Testing improved port freeing function..."
free_port 8080
free_port 8081
free_port 8082
free_port 3000
