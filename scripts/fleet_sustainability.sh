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

print_header() {
    if [ -t 1 ]; then
        clear 2>/dev/null || printf "\033c"
    fi
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}  Fleet Sustainability Manager${NC}"
    echo -e "${BLUE}================================${NC}"
}

# Function to check if a port is in use
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
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

# Function to free a specific port
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
            
            # Kill the process (including Docker processes)
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

# Function to force kill Docker processes blocking ports
kill_docker_processes() {
    print_status "Force killing Docker Desktop and Docker Engine..."
    echo ""
    
    # First, stop all Docker containers
    print_status "Stopping all Docker containers..."
    if docker ps -q 2>/dev/null | grep -q .; then
        print_status "Found running containers, stopping them..."
        docker stop $(docker ps -q) 2>/dev/null || true
        sleep 2
        print_status "Containers stopped"
    else
        print_status "No running containers found"
    fi
    
    # Find and kill the actual Docker Desktop processes by PID
    print_status "Finding and killing Docker Desktop processes..."
    local docker_pids=$(ps aux | grep -E "(Docker Desktop|Docker Desktop Helper)" | grep -v grep | awk '{print $2}')
    
    if [ -n "$docker_pids" ]; then
        print_status "Found Docker processes: $docker_pids"
        for pid in $docker_pids; do
            local process_name=$(ps -p $pid -o comm= 2>/dev/null)
            local process_user=$(ps -p $pid -o user= 2>/dev/null)
            print_status "Force killing $process_name (PID: $pid, User: $process_user)"
            if [ "$process_user" = "root" ]; then
                sudo kill -9 $pid 2>/dev/null || true
            else
                kill -9 $pid 2>/dev/null || true
            fi
        done
    fi
    
    # Kill Docker Desktop application by exact process names
    print_status "Force killing Docker Desktop application processes..."
    killall -9 "Docker Desktop" 2>/dev/null || true
    killall -9 "Docker Desktop Helper" 2>/dev/null || true
    killall -9 "Docker Desktop Helper (GPU)" 2>/dev/null || true
    killall -9 "Docker Desktop Helper (Renderer)" 2>/dev/null || true
    
    # Kill Docker daemon processes
    print_status "Killing Docker daemon processes..."
    sudo pkill -9 -f "com.docker.vmnetd" 2>/dev/null || true
    sudo pkill -9 -f "com.docker.backend" 2>/dev/null || true
    sudo pkill -9 -f "com.docker.hyperkit" 2>/dev/null || true
    
    # Stop Docker daemon services
    print_status "Stopping Docker daemon services..."
    sudo launchctl stop com.docker.docker 2>/dev/null || true
    sudo launchctl stop com.docker.hyperkit 2>/dev/null || true
    sudo launchctl stop com.docker.vmnetd 2>/dev/null || true
    sudo launchctl stop com.docker.backend 2>/dev/null || true
    
    # Kill processes using Docker socket
    print_status "Killing processes using Docker socket..."
    sudo lsof -t /Users/yilmazu/.docker/run/docker.sock 2>/dev/null | xargs sudo kill -9 2>/dev/null || true
    
    # Wait for processes to die
    sleep 3
    
    # Nuclear option - kill everything Docker-related
    print_status "Nuclear option - killing all Docker-related processes..."
    pkill -9 -f "Docker" 2>/dev/null || true
    sudo pkill -9 -f "com.docker" 2>/dev/null || true
    sudo pkill -9 -f "hyperkit" 2>/dev/null || true
    sudo pkill -9 -f "vmnetd" 2>/dev/null || true
    
    # Wait again
    sleep 3
    
    # Check if Docker processes are still running
    local remaining_docker=$(ps aux | grep -E "(Docker Desktop|com\.docker)" | grep -v grep | wc -l)
    if [ "$remaining_docker" -gt 0 ]; then
        print_warning "Docker processes still running, trying final approach..."
        # Final attempt - kill by full path
        pkill -9 -f "/Applications/Docker.app" 2>/dev/null || true
        pkill -9 -f "Docker Desktop" 2>/dev/null || true
        sleep 2
    fi
    
    # Final check
    local final_check=$(ps aux | grep -E "(Docker Desktop|com\.docker)" | grep -v grep | wc -l)
    if [ "$final_check" -gt 0 ]; then
        print_error "❌ Docker processes still running after all attempts"
        print_status "Remaining processes:"
        ps aux | grep -E "(Docker Desktop|com\.docker)" | grep -v grep
        print_status "Try manually: sudo pkill -9 -f 'Docker Desktop'"
    else
        print_status "✅ Docker Desktop and Engine successfully stopped"
    fi
    
    echo ""
}

# Function to check Docker containers
check_docker_containers() {
    print_status "Checking Docker containers..."
    echo ""
    
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running"
        return 1
    fi
    
    print_status "Docker containers status:"
    docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep fleet-sustainability || print_warning "No fleet-sustainability containers found"
    
    echo ""
    print_status "Docker compose status:"
    cd "$(dirname "$(realpath "${BASH_SOURCE[0]:-$0}" 2>/dev/null || echo "${BASH_SOURCE[0]:-$0}")")/.."
    docker-compose ps
}

# Function to check application logs
check_application_logs() {
    print_status "Checking application logs..."
    echo ""
    
    cd "$(dirname "$(realpath "${BASH_SOURCE[0]:-$0}" 2>/dev/null || echo "${BASH_SOURCE[0]:-$0}")")/.."
    
    # Check Docker container logs
    print_status "Backend container logs (last 20 lines):"
    docker-compose logs --tail=20 app 2>/dev/null || print_warning "Backend container not found or not running"
    
    echo ""
    print_status "MongoDB container logs (last 10 lines):"
    docker-compose logs --tail=10 mongo 2>/dev/null || print_warning "MongoDB container not found or not running"
    
    echo ""
    print_status "Mongo Express container logs (last 10 lines):"
    docker-compose logs --tail=10 mongo-express 2>/dev/null || print_warning "Mongo Express container not found or not running"
    
    # Check frontend logs if they exist
    if [ -f "frontend/frontend.log" ]; then
        echo ""
        print_status "Frontend logs (last 20 lines):"
        tail -20 frontend/frontend.log
    else
        print_warning "Frontend log file not found"
    fi
}

# Function to show troubleshooting menu
troubleshooting() {
    while true; do
        print_header
        echo ""
        echo "🔧 Troubleshooting Menu"
        echo ""
        echo "1) Free ports (8080, 8081, 8082, 3000)"
        echo "2) Kill all Docker containers and processes"
        echo "3) Check Docker containers"
        echo "4) Check application logs"
        echo "5) Auto-fix (reset + seed + start movement)"
        echo "6) Back to main menu"
        echo ""
        read -p "Enter your choice (1-6): " choice
        
        case $choice in
            1)
                print_header
                print_status "Freeing up ports 8080, 8081, 8082, and 3000..."
                echo ""
                
                # Free port 8080 (backend internal)
                free_port 8080
                
                # Free port 8081 (backend external)
                free_port 8081
                
                # Free port 8082 (Mongo Express)
                free_port 8082
                
                # Free port 3000 (frontend)
                free_port 3000
                
                echo ""
                print_status "🎉 Port freeing complete!"
                echo ""
                read -p "Press Enter to continue..."
                ;;
            2)
                kill_docker_processes
                echo ""
                read -p "Press Enter to continue..."
                ;;
            3)
                print_header
                check_docker_containers
                echo ""
                read -p "Press Enter to continue..."
                ;;
            4)
                print_header
                check_application_logs
                echo ""
                read -p "Press Enter to continue..."
                ;;
            5)
                print_header
                auto_fix
                echo ""
                read -p "Press Enter to continue..."
                ;;
            6)
                break
                ;;
            *)
                print_error "Invalid choice. Please enter 1-6."
                sleep 2
                ;;
        esac
    done
}

# Simulator submenu
simulator_menu() {
    while true; do
        print_header
        echo ""
        echo "🛰️ Simulator Menu"
        echo ""
        echo "1) Start simulator (local cities)"
        echo "2) Start simulator (global/worldwide)"
        echo "3) Stop simulator"
        echo "4) Simulator status"
        echo "5) Back to main menu"
        echo ""
        read -p "Enter your choice (1-5): " choice
        case $choice in
            1)
                start_simulator local
                echo ""
                read -p "Press Enter to continue..."
                ;;
            2)
                start_simulator global
                echo ""
                read -p "Press Enter to continue..."
                ;;
            3)
                stop_simulator
                echo ""
                read -p "Press Enter to continue..."
                ;;
            4)
                simulator_status
                echo ""
                read -p "Press Enter to continue..."
                ;;
            5)
                break
                ;;
            *)
                print_error "Invalid choice. Please enter 1-5."
                sleep 2
                ;;
        esac
    done
}

# OSRM submenu
osrm_menu() {
    while true; do
        print_header
        echo ""
        echo "🗺️ OSRM Menu"
        echo ""
        echo "1) Start local OSRM"
        echo "2) Stop local OSRM"
        echo "3) OSRM status"
        echo "4) Back to main menu"
        echo ""
        read -p "Enter your choice (1-4): " choice
        case $choice in
            1)
                start_local_osrm
                echo ""
                read -p "Press Enter to continue..."
                ;;
            2)
                stop_local_osrm
                echo ""
                read -p "Press Enter to continue..."
                ;;
            3)
                osrm_status
                echo ""
                read -p "Press Enter to continue..."
                ;;
            4)
                break
                ;;
            *)
                print_error "Invalid choice. Please enter 1-4."
                sleep 2
                ;;
        esac
    done
}

# Function to start the application
start_fleet_sustainability() {
    print_header
    print_status "Starting Fleet Sustainability Application..."
    echo ""

    # Check if Docker is running
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker first."
        exit 1
    fi

    # Start all services with Docker Compose
    print_status "1. Starting all services with Docker Compose..."
    cd "$(dirname "$(realpath "${BASH_SOURCE[0]:-$0}" 2>/dev/null || echo "${BASH_SOURCE[0]:-$0}")")/.."
    docker-compose up -d
    if [ $? -eq 0 ]; then
        print_status "   Docker services started successfully"
    else
        print_error "   Failed to start Docker services"
        exit 1
    fi

    # Wait for services to be ready
    print_status "2. Waiting for services to be ready..."
    sleep 10

    # Start local OSRM
    print_status "5. Starting local OSRM..."
    start_local_osrm || print_warning "Local OSRM could not be started."

    # Check if backend container is running
    print_status "3. Checking backend container..."
    if docker ps | grep -q "fleet-sustainability-app"; then
        print_status "   Backend container is running"
    else
        print_error "   Backend container is not running"
        print_status "   Checking logs..."
        docker-compose logs app
        exit 1
    fi

    # Check if MongoDB container is running
    print_status "4. Checking MongoDB container..."
    if docker ps | grep -q "fleet-sustainability-mongo"; then
        print_status "   MongoDB container is running"
    else
        print_error "   MongoDB container is not running"
        exit 1
    fi

    # Ensure admin user exists
    print_status "5. Ensuring admin user exists..."
    ensure_admin_user
    if [ $? -eq 0 ]; then
        print_status "   Admin user is ready"
    else
        print_warning "   Admin user creation had issues, but continuing..."
    fi

    # Start frontend (local process)
    print_status "6. Starting frontend..."
    
    # Kill any existing frontend process
    kill_port 3000
    
    # Start frontend in background
    cd "$(dirname "$(realpath "${BASH_SOURCE[0]:-$0}" 2>/dev/null || echo "${BASH_SOURCE[0]:-$0}")")/.."
    cd frontend
    npm start > frontend.log 2>&1 &
    FRONTEND_PID=$!
    
    # Wait for frontend to start
    print_status "   Waiting for frontend to start..."
    for i in {1..30}; do
        if check_port 3000; then
            print_status "   Frontend started successfully on port 3000"
            break
        fi
        sleep 1
    done
    
    if ! check_port 3000; then
        print_error "   Frontend failed to start"
        exit 1
    fi

    # Save frontend PID to file for later cleanup
    cd "$(dirname "$(realpath "${BASH_SOURCE[0]:-$0}" 2>/dev/null || echo "${BASH_SOURCE[0]:-$0}")")/.."
    echo "$FRONTEND_PID" > .frontend_pid

    # Test the application
    print_status "7. Testing application..."
    sleep 3
    
    # Ensure admin user exists for testing
    ensure_admin_user
    
    # Test login API (backend is now on port 8081 via Docker)
    LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8081/api/auth/login \
        -H "Content-Type: application/json" \
        -d '{"username": "admin", "password": "admin123"}')
    
    if echo "$LOGIN_RESPONSE" | grep -q "token"; then
        print_status "   Login API test passed"
    else
        print_warning "   Login API test failed"
        print_status "   Checking backend logs..."
        docker-compose logs app
    fi

    echo ""
    print_status "🎉 Fleet Sustainability is now running!"
    echo ""
    echo -e "${BLUE}Access Points:${NC}"
    echo "   🌐 Frontend: http://localhost:3000"
    echo "   🔧 Backend API: http://localhost:8081"
    echo "   🗄️  Mongo Express: http://localhost:8082"
    echo ""
    echo -e "${BLUE}Login Credentials:${NC}"
    echo "   👤 Username: admin"
    echo "   🔑 Password: admin123"
    echo ""
    echo -e "${BLUE}Other Users:${NC}"
    echo "   👤 manager / manager123"
    echo "   👤 operator / operator123"
    echo "   👤 viewer / viewer123"
    echo ""
    echo -e "${YELLOW}To stop the application, run:${NC}"
    echo "   ./scripts/fleet_sustainability.sh stop"
    echo ""
    
    # Open browser
    print_status "Opening browser..."
    open http://localhost:3000
}

# Function to stop the application
stop_fleet_sustainability() {
    print_header
    print_status "Stopping Fleet Sustainability Application..."
    echo ""

    # Stop frontend
    print_status "1. Stopping frontend..."
    kill_port 3000
    if [ -f ".frontend_pid" ]; then
        FRONTEND_PID=$(cat .frontend_pid)
        kill -9 $FRONTEND_PID 2>/dev/null
        rm -f .frontend_pid
    fi
    print_status "   Frontend stopped"

    # Stop Docker containers
    print_status "2. Stopping Docker containers..."
    cd "$(dirname "$(realpath "${BASH_SOURCE[0]:-$0}" 2>/dev/null || echo "${BASH_SOURCE[0]:-$0}")")/.."
    docker-compose down
    print_status "   Docker containers stopped"

    # Stop local OSRM
    print_status "3. Stopping local OSRM..."
    stop_local_osrm

    # Clean up log files
    print_status "4. Cleaning up..."
    rm -f backend.log frontend/frontend.log 2>/dev/null

    echo ""
    print_status "✅ Fleet Sustainability stopped successfully!"
    echo ""
}

# Function to show status
show_status() {
    print_header
    print_status "Checking Fleet Sustainability Status..."
    echo ""

    # Check Docker containers
    print_status "Docker Containers:"
    if docker ps | grep -q "fleet-sustainability-app"; then
        print_status "✅ Backend: Running (Docker container)"
    else
        print_warning "❌ Backend: Not running (Docker container)"
    fi

    if docker ps | grep -q "fleet-sustainability-mongo"; then
        print_status "✅ MongoDB: Running (Docker container)"
    else
        print_warning "❌ MongoDB: Not running (Docker container)"
    fi

    if docker ps | grep -q "fleet-sustainability-mongo-express"; then
        print_status "✅ Mongo Express: Running (Docker container)"
    else
        print_warning "❌ Mongo Express: Not running (Docker container)"
    fi

    if docker ps | grep -q "fleet-osrm"; then
        print_status "✅ OSRM: Running (Docker container)"
    else
        print_warning "❌ OSRM: Not running (Docker container)"
    fi

    # Check frontend
    if check_port 3000; then
        print_status "✅ Frontend: Running on port 3000"
    else
        print_warning "❌ Frontend: Not running"
    fi

    # Check API endpoints
    print_status "API Endpoints:"
    if curl -s http://localhost:8081 > /dev/null 2>&1; then
        print_status "✅ Backend API: Responding on port 8081"
    else
        print_warning "❌ Backend API: Not responding on port 8081"
    fi

    if curl -s http://localhost:8082 > /dev/null 2>&1; then
        print_status "✅ Mongo Express: Responding on port 8082"
    else
        print_warning "❌ Mongo Express: Not responding on port 8082"
    fi

    # OSRM visibility
    print_status "Routing (OSRM):"
    osrm_status

    echo ""
    # Pause in interactive sessions so the screen doesn't clear immediately
    if [ -t 0 ]; then
        read -p "Press Enter to go back..." </dev/tty
    fi
}

# Function to show help
show_help() {
    print_header
    echo ""
    echo "Usage: $0 [OPTION]"
    echo ""
    echo "Options:"
    echo "  start         Start the Fleet Sustainability application"
    echo "  stop          Stop the Fleet Sustainability application"
    echo "  status        Show the status of all services"
    echo "  restart       Restart the application (stop then start)"
    echo "  populate      Populate database with dummy data"
    echo "  clear         Clear database data (preserves users)"
    echo "  sim-start     Start simulator (vehicles moving) [env: FLEET_SIZE, SIM_GLOBAL=1]"
    echo "  sim-stop      Stop simulator"
    echo "  sim-status    Show simulator status"
    echo "  auto-fix      Auto-fix: reset DB, seed, and start movement"
    echo "  osrm-start    Start local OSRM (Monaco dataset) on http://localhost:5000"
    echo "  osrm-stop     Stop local OSRM"
    echo "  osrm-status   Check OSRM reachability (and local container state)"
    echo "  troubleshoot  Open troubleshooting menu"
    echo "  help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  RUN_LOCAL_OSRM=1 $0 start   # start app and local OSRM"
    echo "  $0 sim-start                 # local cities (uses OSRM_BASE_URL if set)"
    echo "  $0 sim-start global          # force global spawn + public OSRM"
    echo "  OSRM_BASE_URL=https://router.project-osrm.org SIM_GLOBAL=1 $0 sim-start"
    echo ""
    # Final success message
    print_status "✅ Populate completed successfully!"
    print_status "   Created $VEH_CREATED vehicles with $TRIP_CREATED trips"
    print_status "   Simulator: Running with all vehicles using global OSRM"
    print_status "   Vehicles should now be moving and snapped to roads!"
    
    # Pause in interactive sessions so the screen doesn't clear immediately
    if [ -t 0 ]; then
        read -p "Press Enter to go back..." </dev/tty
    fi
}

# Function to clear database data (safe tables only)
clear_database() {
    print_header
    print_status "Clearing database data (preserving users)..."
    echo ""

    # Stop containers to ensure clean state
    print_status "1. Stopping containers..."
    docker-compose down > /dev/null 2>&1
    print_status "   Containers stopped"

    # Start containers fresh
    print_status "2. Starting containers..."
    docker-compose up -d > /dev/null 2>&1
    print_status "   Containers started"

    # Wait for backend to be ready
    print_status "3. Waiting for backend to be ready..."
    for i in {1..30}; do
        if curl -s http://localhost:8081 > /dev/null 2>&1; then
            print_status "   Backend is ready"
            break
        fi
        if [ $i -eq 30 ]; then
            print_error "Backend failed to start within 30 seconds"
            exit 1
        fi
        sleep 1
    done

    # Ensure admin user exists
    print_status "4. Ensuring admin user exists..."
    ensure_admin_user

    # Get admin token for API calls
    print_status "5. Getting admin authentication token..."
    LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8081/api/auth/login \
        -H "Content-Type: application/json" \
        -d '{"username": "admin", "password": "admin123"}')
    
    if ! echo "$LOGIN_RESPONSE" | grep -q "token"; then
        print_error "Failed to authenticate as admin. Please ensure admin user exists."
        print_status "   Response: $LOGIN_RESPONSE"
        exit 1
    fi
    
    # Extract token more reliably using jq if available, otherwise use grep
    if command -v jq >/dev/null 2>&1; then
        TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token')
    else
        TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    fi
    
    if [ -z "$TOKEN" ]; then
        print_error "Failed to extract token from response"
        print_status "   Response: $LOGIN_RESPONSE"
        exit 1
    fi
    
    print_status "   Authentication successful"

    # Clear telemetry data to prevent old data from affecting metrics
    print_status "6. Clearing telemetry data..."
    TELE_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/telemetry || echo 000)
    if [ "$TELE_CODE" = "200" ] || [ "$TELE_CODE" = "204" ]; then
        print_status "   Telemetry data cleared successfully (API)"
    else
        print_warning "   API delete not available (HTTP $TELE_CODE); attempting DB fallback..."
        CLEARED=0
        if command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' | grep -q '^fleet-sustainability-mongo$'; then
            docker exec fleet-sustainability-mongo mongosh --quiet --username root --password example --eval 'db.getSiblingDB("fleet").telemetry.deleteMany({})' >/dev/null 2>&1 && CLEARED=1
        fi
        if [ "$CLEARED" -ne 1 ] && command -v mongosh >/dev/null 2>&1; then
            mongosh --quiet 'mongodb://localhost:27017' --eval 'db.getSiblingDB("fleet").telemetry.deleteMany({})' >/dev/null 2>&1 && CLEARED=1
        fi
        if [ "$CLEARED" -eq 1 ]; then
            print_status "   Telemetry data cleared successfully (DB)"
        else
            print_warning "   Failed to clear telemetry data via DB fallback"
        fi
    fi

    # Global city centers used for seeding locations (subset to keep runtime reasonable)
    CITIES=(
        "35.6895:139.6917"   # Tokyo
        "28.6139:77.2090"    # Delhi
        "31.2304:121.4737"   # Shanghai
        "40.7128:-74.0060"   # New York
        "-23.5505:-46.6333"  # São Paulo
    )

    # Clear vehicles by listing and deleting individually (bulk delete not supported)
    print_status "7. Clearing vehicles..."
    LIST=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/vehicles)
    IDS=()
    if command -v jq >/dev/null 2>&1; then
        while IFS= read -r id; do [ -n "$id" ] && IDS+=("$id"); done < <(echo "$LIST" | jq -r '.[]? | .id // empty')
    else
        while IFS= read -r id; do [ -n "$id" ] && IDS+=("$id"); done < <(echo "$LIST" | grep -o '"id":"[a-f0-9]\{24\}"' | cut -d '"' -f4)
    fi
    TOTAL=${#IDS[@]}
    if [ "$TOTAL" -gt 0 ]; then
        CNT=0
        for id in "${IDS[@]}"; do
            CNT=$((CNT+1))
            progress_print "   Deleting vehicles:" "$CNT" "$TOTAL"
            curl -s -X DELETE -H "Authorization: Bearer $TOKEN" "http://localhost:8081/api/vehicles/$id" >/dev/null 2>&1 || true
        done
        progress_done "   Deleting vehicles:" "$CNT" "$TOTAL"
        print_status "   Vehicles cleared: $CNT"
    else
        print_status "   No vehicles to clear"
    fi

    # Clear telemetry (API or DB fallback)
    print_status "8. Clearing telemetry data..."
    TELE_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/telemetry || echo 000)
    if [ "$TELE_CODE" = "200" ] || [ "$TELE_CODE" = "204" ]; then
        print_status "   Telemetry data cleared successfully (API)"
    else
        print_warning "   API delete not available (HTTP $TELE_CODE); attempting DB fallback..."
        CLEARED=0
        if command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' | grep -q '^fleet-sustainability-mongo$'; then
            docker exec fleet-sustainability-mongo mongosh --quiet --username root --password example --eval 'db.getSiblingDB("fleet").telemetry.deleteMany({})' >/dev/null 2>&1 && CLEARED=1
        fi
        if [ "$CLEARED" -ne 1 ] && command -v mongosh >/dev/null 2>&1; then
            mongosh --quiet 'mongodb://localhost:27017' --eval 'db.getSiblingDB("fleet").telemetry.deleteMany({})' >/dev/null 2>&1 && CLEARED=1
        fi
        if [ "$CLEARED" -eq 1 ]; then
            print_status "   Telemetry data cleared successfully (DB)"
        else
            print_warning "   Failed to clear telemetry data via DB fallback"
        fi
    fi

    # Clear trips using API DELETE endpoint
    print_status "8. Clearing trips..."
    TRIPS_RESPONSE=$(curl -s -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/trips)
    if echo "$TRIPS_RESPONSE" | grep -q "deleted successfully"; then
        print_status "   Trips cleared successfully"
    else
        print_warning "   Failed to clear trips: $TRIPS_RESPONSE"
    fi

    # Clear maintenance using API DELETE endpoint
    print_status "9. Clearing maintenance records..."
    MAINTENANCE_RESPONSE=$(curl -s -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/maintenance)
    if echo "$MAINTENANCE_RESPONSE" | grep -q "deleted successfully"; then
        print_status "   Maintenance records cleared successfully"
    else
        print_warning "   Failed to clear maintenance records: $MAINTENANCE_RESPONSE"
    fi

    # Clear costs using API DELETE endpoint
    print_status "10. Clearing cost records..."
    COSTS_RESPONSE=$(curl -s -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/costs)
    if echo "$COSTS_RESPONSE" | grep -q "deleted successfully"; then
        print_status "   Cost records cleared successfully"
    else
        print_warning "   Failed to clear cost records: $COSTS_RESPONSE"
    fi

    echo ""
    print_status "🎉 Database cleared successfully!"
    echo ""
    echo -e "${BLUE}Cleared:${NC}"
    echo "   🚗 All vehicles"
    echo "   📊 All telemetry records"
    echo "   🛣️  All trips"
    echo "   🔧 All maintenance records"
    echo "   💰 All cost records"
    echo ""
    echo -e "${YELLOW}Preserved:${NC}"
    echo "   👤 User accounts (admin, manager, operator, viewer)"
    echo "   🔐 Authentication data"
    echo ""
    echo -e "${YELLOW}You can now run populate to add fresh data!${NC}"
    echo ""
}

# --- Progress helpers ---
progress_draw() {
    local current=$1
    local total=$2
    local width=${3:-40}
    if [ "$total" -le 0 ]; then total=1; fi
    local percent=$(( current * 100 / total ))
    local filled=$(( current * width / total ))
    local empty=$(( width - filled ))
    printf "[%.*s%*s] %3d%% (%d/%d)" "$filled" "########################################" "$empty" "" "$percent" "$current" "$total"
}

progress_print() {
    local label="$1"; shift
    local current=$1; local total=$2
    echo -ne "${label} "
    progress_draw "$current" "$total"
    echo -ne "\r"
}

progress_done() {
    local label="$1"; shift
    local current=$1; local total=$2
    echo -ne "${label} "
    progress_draw "$current" "$total"
    echo -e "\n"
}
# --- End progress helpers ---

# --- Populate helpers ---
choose_window() {
    echo ""
    echo "Select time window:"
    echo "  1) 1 hour"
    echo "  2) 1 day"
    echo "  3) 1 week"
    echo "  4) 1 month"
    read -p "Enter choice (1-4) [2]: " wch
    case "${wch:-2}" in
        1) WINDOW=1h ; STEP_SECONDS=${STEP_SECONDS:-15} ;;
        2) WINDOW=24h ; STEP_SECONDS=${STEP_SECONDS:-15} ;;
        3) WINDOW=7d  ; STEP_SECONDS=${STEP_SECONDS:-30} ;;
        4) WINDOW=30d ; STEP_SECONDS=${STEP_SECONDS:-60} ;;
        *) WINDOW=24h ; STEP_SECONDS=${STEP_SECONDS:-15} ;;
    esac
    print_status "   Window selected: $WINDOW"
}
# --- End populate helpers ---

# Function to populate database with dummy data
populate_database() {
    print_header
    print_status "Populating database with dummy data..."
    echo ""

    # Check if backend is running
    if ! curl -s http://localhost:8081 > /dev/null 2>&1; then
        print_error "Backend is not running. Please start the application first."
        exit 1
    fi

    # Ensure admin user exists first
    print_status "1. Ensuring admin user exists..."
    ensure_admin_user
    if [ $? -ne 0 ]; then
        print_error "Failed to ensure admin user exists. Cannot proceed with population."
        print_status "   Please start the application first with: ./scripts/fleet_sustainability.sh start"
        exit 1
    fi

    # Get admin token for API calls
    print_status "2. Getting admin authentication token..."
    LOGIN_RESPONSE=$(curl -s -m 5 -X POST http://localhost:8081/api/auth/login \
        -H "Content-Type: application/json" \
        -d '{"username": "admin", "password": "admin123"}')
    
    if ! echo "$LOGIN_RESPONSE" | grep -q "token"; then
        print_error "Failed to authenticate as admin. Please ensure admin user exists."
        print_status "   Response: $LOGIN_RESPONSE"
        exit 1
    fi
    
    # Extract token more reliably using jq if available, otherwise use grep
    if command -v jq >/dev/null 2>&1; then
        TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token')
    else
        TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    fi
    
    if [ -z "$TOKEN" ]; then
        print_error "Failed to extract token from response"
        print_status "   Response: $LOGIN_RESPONSE"
        exit 1
    fi
    
    print_status "   Authentication successful"

    # Sanity check API reachability
    API_CODE=$(curl -s -o /dev/null -w '%{http_code}' -m 5 -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/vehicles)
    if [ "$API_CODE" != "200" ]; then
        print_warning "   API health check failed (HTTP $API_CODE). Telemetry import may retry auth on 401."
    fi

    # Populate wizard
    echo ""
    echo "Populate options:"
    echo "  0) Quick Test (1 vehicle, ~60 points over 10 minutes)"
    echo "  1) Basic (top 5 cities, ~1000 telemetry points total)"
    echo "  2) City-based (select cities and window)"
    echo "  3) Quick Commute (1 vehicle, commute profile: 10→50 km/h→10, park)"
    if [ "${MODE_QUICK:-0}" = "1" ]; then
        psel=0
        print_status "   Non-interactive quick mode selected"
    else
        read -p "Choose option (0-2) [1]: " psel
        psel=${psel:-1}
    fi

    # Default cap lowered to 1000 for faster runs
    MAX_TELE_POINTS=${MAX_TELE_POINTS:-1000}

    if [ "$psel" = "0" ]; then
        QUICK=1
        # Focus on one well-known city (NYC)
        CITIES=(
            "40.7128:-74.0060"
        )
        # Quick window params are handled later; skip choose_window
        print_status "   Quick Test selected: 1 vehicle, ~60 points, ~10 minutes"
    elif [ "$psel" = "3" ]; then
        # Commute profile with selectable window (not quick: we want longer ranges)
        QUICK=0
        COMMUTE_MODE=1
        # NYC default
        CITIES=(
            "40.7128:-74.0060"
        )
        print_status "   Commute profile selected"
        choose_window
    elif [ "$psel" = "2" ]; then
        # Minimal catalog (you can extend this list)
        CITY_CATALOG=(
            "51.5074:-0.1278|United Kingdom|London"
            "40.7128:-74.0060|USA|New York"
            "48.8566:2.3522|France|Paris"
            "41.0082:28.9784|Türkiye|Istanbul"
            "35.6895:139.6917|Japan|Tokyo"
            "34.0522:-118.2437|USA|Los Angeles"
            "28.6139:77.2090|India|Delhi"
            "31.2304:121.4737|China|Shanghai"
            "-23.5505:-46.6333|Brazil|São Paulo"
            "30.0444:31.2357|Egypt|Cairo"
        )
        echo ""
        echo "Available cities:"
        idx=0
        for rec in "${CITY_CATALOG[@]}"; do
            idx=$((idx+1))
            latlon="${rec%%|*}"; rest="${rec#*|}"; country="${rest%%|*}"; city="${rest##*|}"
            printf "  %2d) %-18s  %-18s  %s\n" "$idx" "$latlon" "$country" "$city"
        done
        echo ""
        read -p "Enter selections (comma-separated indexes) or press Enter for top 5: " sel
        CITIES=()
        if [ -n "$sel" ]; then
            IFS=',' read -r -a picks <<< "$sel"
            for p in "${picks[@]}"; do
                ptrim=$(echo "$p" | tr -d ' ')
                if [[ "$ptrim" =~ ^[0-9]+$ ]]; then
                    ii=$((ptrim-1))
                    if [ $ii -ge 0 ] && [ $ii -lt ${#CITY_CATALOG[@]} ]; then
                        latlon="${CITY_CATALOG[$ii]%%|*}"
                        CITIES+=("$latlon")
                    fi
                fi
            done
        fi
        if [ ${#CITIES[@]} -eq 0 ]; then
            CITIES=(
                "35.6895:139.6917"
                "28.6139:77.2090"
                "31.2304:121.4737"
                "40.7128:-74.0060"
                "-23.5505:-46.6333"
            )
        fi
        
        # Convert cities array to SIM_EXTRA_CITIES format for simulator
        SIM_EXTRA_CITIES=""
        for city in "${CITIES[@]}"; do
            if [ -n "$SIM_EXTRA_CITIES" ]; then
                SIM_EXTRA_CITIES="$SIM_EXTRA_CITIES;"
            fi
            # Convert "lat:lon" to "lat,lon" format
            latlon=$(echo "$city" | tr ':' ',')
            SIM_EXTRA_CITIES="$SIM_EXTRA_CITIES$latlon"
        done
        
        print_status "Selected cities for simulator: ${CITIES[*]}"
        print_status "SIM_EXTRA_CITIES will be set to: $SIM_EXTRA_CITIES"
        
        # Export the variable so it's available when simulator starts
        export SIM_EXTRA_CITIES
        
        choose_window
    else
        # Basic: top 5 cities and window prompt
        CITIES=(
            "35.6895:139.6917"
            "28.6139:77.2090"
            "31.2304:121.4737"
            "40.7128:-74.0060"
            "-23.5505:-46.6333"
        )
        choose_window
    fi

    # If CITIES not set by wizard (fallback safety)
    if [ ${#CITIES[@]} -eq 0 ]; then
        CITIES=(
            "35.6895:139.6917"
            "28.6139:77.2090"
            "31.2304:121.4737"
            "40.7128:-74.0060"
            "-23.5505:-46.6333"
        )
    fi

    # Create dummy vehicles - quick or full set
    print_status "3. Creating dummy vehicles..."
    VEHICLE_IDS=()
    VEHICLE_TYPES=()
    if [ "${QUICK:-0}" = "1" ]; then
        # One EV near selected city (default NYC)
        CITY_COORDS="${CITIES[0]}"
        BASE_LAT=$(echo "$CITY_COORDS" | cut -d':' -f1)
        BASE_LON=$(echo "$CITY_COORDS" | cut -d':' -f2)
        V_LAT=$(echo "$BASE_LAT" | awk '{printf "%.6f", $1}')
        V_LON=$(echo "$BASE_LON" | awk '{printf "%.6f", $1}')
        VEHICLE_BODY=$(cat <<JSON
{"type":"EV","make":"Tesla","model":"Model 3","year":2023,"status":"active","current_location":{"lat":$V_LAT,"lon":$V_LON}}
JSON
)
        RESP=$(curl -s -X POST http://localhost:8081/api/vehicles \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $TOKEN" \
            -d "$VEHICLE_BODY")
        VID=""
        if command -v jq >/dev/null 2>&1; then
            VID=$(echo "$RESP" | jq -r '.id // empty')
        else
            VID=$(echo "$RESP" | grep -o '"id":"[a-f0-9]\{24\}"' | cut -d '"' -f4)
        fi
        if [ -n "$VID" ]; then
            VEHICLE_IDS+=("$VID")
            QUICK_VID="$VID"
            progress_done "   Creating vehicles:" 1 1
            print_status "   Vehicles created: total=1 (EV=1, ICE=0)"
        else
            print_error "   Failed to create vehicle: $RESP"
            return 1
        fi
    else
    VEHICLES=(
        '{"type": "EV", "make": "Tesla", "model": "Model 3", "year": 2023, "status": "active"}'
        '{"type": "EV", "make": "Tesla", "model": "Model Y", "year": 2023, "status": "active"}'
        '{"type": "ICE", "make": "Ford", "model": "F-150", "year": 2022, "status": "active"}'
        '{"type": "ICE", "make": "Toyota", "model": "Tacoma", "year": 2022, "status": "active"}'
    )
        TOTAL_VEH=${#VEHICLES[@]}
        CREATED=0
        EV_CREATED=0
        ICE_CREATED=0
        IDX=0
    for vehicle in "${VEHICLES[@]}"; do
            IDX=$((IDX+1))
            progress_print "   Creating vehicles:" "$IDX" "$TOTAL_VEH"
            CITY_IDX=$((RANDOM % ${#CITIES[@]}))
            CITY_COORDS="${CITIES[$CITY_IDX]}"
            BASE_LAT=$(echo "$CITY_COORDS" | cut -d':' -f1)
            BASE_LON=$(echo "$CITY_COORDS" | cut -d':' -f2)
            
            # Generate land-validated coordinates (avoid sea areas)
            ATTEMPTS=0
            MAX_ATTEMPTS=10
            while [ $ATTEMPTS -lt $MAX_ATTEMPTS ]; do
                LAT_OFF=$(echo "scale=4; ($RANDOM % 100 - 50) / 1000" | bc -l 2>/dev/null || echo "0.0200")
                LON_OFF=$(echo "scale=4; ($RANDOM % 100 - 50) / 1000" | bc -l 2>/dev/null || echo "0.0200")
                V_LAT=$(echo "scale=6; $BASE_LAT + $LAT_OFF" | bc -l 2>/dev/null || echo "$BASE_LAT")
                V_LON=$(echo "scale=6; $BASE_LON + $LON_OFF" | bc -l 2>/dev/null || echo "$BASE_LON")
                
                # Simple land validation: avoid extreme coordinates that are likely sea
                # This is a basic check - in production you'd use a proper land/sea API
                if (( $(echo "$V_LAT > -60 && $V_LAT < 80" | bc -l) )) && (( $(echo "$V_LON > -180 && $V_LON < 180" | bc -l) )); then
                    # Additional check: avoid coordinates that look like they're in the middle of oceans
                    # This is a simplified check - real implementation would use a land/sea database
                    LAT_ABS=$(echo "scale=6; if($V_LAT < 0) -$V_LAT else $V_LAT" | bc -l)
                    LON_ABS=$(echo "scale=6; if($V_LON < 0) -$V_LON else $V_LON" | bc -l)
                    if (( $(echo "$LAT_ABS > 0.1 && $LON_ABS > 0.1" | bc -l) )); then
                        break  # Valid land coordinates found
                    fi
                fi
                ATTEMPTS=$((ATTEMPTS+1))
            done
            
            # If we couldn't find good coordinates, use the city center
            if [ $ATTEMPTS -eq $MAX_ATTEMPTS ]; then
                V_LAT="$BASE_LAT"
                V_LON="$BASE_LON"
            fi
            VEHICLE_BODY=$(python3 - <<PY
import json
v=json.loads('''$vehicle''')
v['current_location']={'lat': float('$V_LAT'), 'lon': float('$V_LON')}
print(json.dumps(v))
PY
)
            HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8081/api/vehicles \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $TOKEN" \
                -d "$VEHICLE_BODY")
            if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 300 ]; then
                CREATED=$((CREATED+1))
                if echo "$vehicle" | grep -q '"type": "EV"'; then EV_CREATED=$((EV_CREATED+1)); else ICE_CREATED=$((ICE_CREATED+1)); fi
        fi
    done
        progress_done "   Creating vehicles:" "$CREATED" "$TOTAL_VEH"
        print_status "   Vehicles created: total=$CREATED (EV=$EV_CREATED, ICE=$ICE_CREATED)"
        # Refresh vehicle list and extract IDs and types for subsequent data creation
    VEHICLES_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/vehicles)
        if command -v jq >/dev/null 2>&1; then
            while read -r id type; do
                [ -n "$id" ] && VEHICLE_IDS+=("$id") && VEHICLE_TYPES+=("$type")
            done < <(echo "$VEHICLES_RESPONSE" | jq -r '.[]? | "\(.id // empty) \(.type // empty)"')
            if [ ${#VEHICLE_IDS[@]} -eq 0 ]; then 
                while read -r id type; do [ -n "$id" ] && VEHICLE_IDS+=("$id") && VEHICLE_TYPES+=("$type"); done < <(echo "$VEHICLES_RESPONSE" | jq -r '.data[]? | "\(.id // empty) \(.type // empty)"')
            fi
        fi
        if [ ${#VEHICLE_IDS[@]} -eq 0 ]; then 
            # Fallback: extract both ID and type using grep/sed
            while IFS= read -r line; do
                id=$(echo "$line" | grep -o '"id":"[a-f0-9]\{24\}"' | cut -d '"' -f4)
                type=$(echo "$line" | grep -o '"type":"[^"]*"' | cut -d '"' -f4)
                [ -n "$id" ] && VEHICLE_IDS+=("$id") && VEHICLE_TYPES+=("$type")
            done < <(echo "$VEHICLES_RESPONSE" | grep -E '"id":"[a-f0-9]{24}"' | head -10)
        fi
        print_status "   Vehicle IDs detected: ${#VEHICLE_IDS[@]}"
    fi


    # Telemetry time series
    print_status "4. Creating dummy telemetry time series..."
    if [ "${QUICK:-0}" = "1" ]; then
        NOW_EPOCH=$(date -u +%s)
        START_EPOCH=$(( NOW_EPOCH - 600 ))
        STEP_SECONDS=${STEP_SECONDS:-10}
        MAX_TELE_POINTS=${MAX_TELE_POINTS:-120}
    else
        WINDOW=${WINDOW:-24h}
        case "$WINDOW" in
            24h)   START_EPOCH=$(date -u -v-24H +%s) ; STEP_SECONDS=${STEP_SECONDS:-15} ;;
            7d)    START_EPOCH=$(date -u -v-7d +%s)  ; STEP_SECONDS=${STEP_SECONDS:-30} ;;
            30d)   START_EPOCH=$(date -u -v-30d +%s) ; STEP_SECONDS=${STEP_SECONDS:-60} ;;
            *)     START_EPOCH=$(date -u -v-24H +%s) ; STEP_SECONDS=${STEP_SECONDS:-15} ;;
        esac
        NOW_EPOCH=$(date -u +%s)
    fi

    VEH_COUNT=${#VEHICLE_IDS[@]}
    RAW_PER_VEH=$(( (NOW_EPOCH-START_EPOCH) / STEP_SECONDS ))
    if [ "$RAW_PER_VEH" -lt 1 ]; then RAW_PER_VEH=1; fi
    TOTAL_RAW=$(( RAW_PER_VEH * VEH_COUNT ))

    MAX_TELE_POINTS=${MAX_TELE_POINTS:-1000}
    STRIDE=1
    STEP_EFFECTIVE=$STEP_SECONDS
    TOTAL_POINTS=$TOTAL_RAW
    if [ "$TOTAL_RAW" -gt "$MAX_TELE_POINTS" ]; then
        # ceil division to compute stride
        STRIDE=$(( (TOTAL_RAW + MAX_TELE_POINTS - 1) / MAX_TELE_POINTS ))
        if [ "$STRIDE" -lt 1 ]; then STRIDE=1; fi
        STEP_EFFECTIVE=$(( STEP_SECONDS * STRIDE ))
        TOTAL_POINTS=$MAX_TELE_POINTS
        print_status "   Downsampling telemetry: stride=$STRIDE (every $STRIDE step)"
    fi

    TELE_POSTED=0
    EV_POINTS=0
    ICE_POINTS=0
    EV_BATT_UPDATES=0
    ICE_FUEL_UPDATES=0

    if [ ${#VEHICLE_IDS[@]} -eq 0 ]; then
        print_warning "   No vehicle IDs. Skipping telemetry generation."
    else
        print_status "   Posting telemetry: total expected points ~ $TOTAL_POINTS"
        # Helper to print progress periodically
        _telemetry_tick() {
            local label="   Telemetry"
            progress_print "$label:" "$TELE_POSTED" "$TOTAL_POINTS"
        }
        # Show initial 0%
        _telemetry_tick
        LAST_PRINT_TS=$(date +%s)

        # Seed per-vehicle series
        for vid in "${VEHICLE_IDS[@]}"; do
            # Assign type deterministically to ensure consistent EV/ICE split
            if [ $((RANDOM % 100)) -lt 50 ]; then VEHICLE_TYPE=EV; else VEHICLE_TYPE=ICE; fi

            # Start at random location near a city
            CITY_INDEX=$((RANDOM % ${#CITIES[@]}))
            CITY_COORDS="${CITIES[$CITY_INDEX]}"
            BASE_LAT=$(echo "$CITY_COORDS" | cut -d':' -f1)
            BASE_LON=$(echo "$CITY_COORDS" | cut -d':' -f2)
            # Randomize start within ~3km disk around city center to avoid collinear markers
            read LAT LON <<< $(python3 - <<PY
import math,random
base_lat=float("$BASE_LAT"); base_lon=float("$BASE_LON")
R=6378137.0
radius_m=3000.0*random.random()
theta=2*math.pi*random.random()
dlat=(radius_m/R)*(180.0/math.pi)
dlon=(radius_m/(R*max(1e-6,math.cos(math.radians(base_lat)))))*(180.0/math.pi)
lat=base_lat + dlat*math.cos(theta)
lon=base_lon + dlon*math.sin(theta)
print(f"{lat:.6f} {lon:.6f}")
PY
)
            # Initialize a random heading per vehicle
            BEARING=$(python3 - <<PY
import random
print(f"{random.random()*360:.2f}")
PY
)

            # Initial levels
            if [ "$VEHICLE_TYPE" = "EV" ]; then
                FUEL_LEVEL=0
                BATTERY_LEVEL=$((RANDOM % 41 + 60))
            else
                FUEL_LEVEL=$((RANDOM % 41 + 60))
                BATTERY_LEVEL=0
            fi
            SPEED=0

            for ((ts=$START_EPOCH; ts<=$NOW_EPOCH; ts+=$STEP_EFFECTIVE)); do
                if [ "${COMMUTE_MODE:-0}" = "1" ]; then
                    ELAPSED=$((ts-START_EPOCH))
                    # Commute profile timings (seconds)
                    LOCALK=${COMM_LOCAL_KMH:-10}
                    HWK=${COMM_HIGHWAY_KMH:-50}
                    LOCAL_HOLD=${COMM_LOCAL_HOLD_SECS:-300}
                    ACCEL=${COMM_ACCEL_SECS:-120}
                    HW_HOLD=${COMM_HIGHWAY_HOLD_SECS:-300}
                    DECEL=${COMM_DECEL_SECS:-120}
                    LOCAL_HOLD2=${COMM_LOCAL_HOLD_SECS:-300}
                    PARK=${COMM_PARK_SECS:-18000}
                    CYCLE=$((LOCAL_HOLD + ACCEL + HW_HOLD + DECEL + LOCAL_HOLD2 + PARK))
                    TMOD=$((ELAPSED % CYCLE))
                    # Determine target speed by phase
                    if [ $TMOD -lt $LOCAL_HOLD ]; then
                        SPEED=$LOCALK
                    elif [ $TMOD -lt $((LOCAL_HOLD + ACCEL)) ]; then
                        T=$((TMOD - LOCAL_HOLD))
                        SPEED=$(python3 - <<PY
loc=$LOCALK
hw=$HWK
acc=$ACCEL
t=$T
print(int(loc + (hw-loc)*t/acc))
PY
)
                    elif [ $TMOD -lt $((LOCAL_HOLD + ACCEL + HW_HOLD)) ]; then
                        SPEED=$HWK
                    elif [ $TMOD -lt $((LOCAL_HOLD + ACCEL + HW_HOLD + DECEL)) ]; then
                        T=$((TMOD - LOCAL_HOLD - ACCEL - HW_HOLD))
                        SPEED=$(python3 - <<PY
loc=$LOCALK
hw=$HWK
dec=$DECEL
t=$T
print(int(hw - (hw-loc)*t/dec))
PY
)
                    elif [ $TMOD -lt $((LOCAL_HOLD + ACCEL + HW_HOLD + DECEL + LOCAL_HOLD2)) ]; then
                        SPEED=$LOCALK
                    else
                        SPEED=0
                    fi
                else
                    # Motion: 3 steps moving, 1 step idle (approximate with stride)
                    if [ $(((ts-START_EPOCH)/STEP_SECONDS % 4)) -lt 3 ]; then
                        TARGET=$((RANDOM % 60 + 10))
                        if [ $SPEED -lt $TARGET ]; then SPEED=$((SPEED+5)); else SPEED=$((SPEED-3)); fi
                        if [ $SPEED -lt 0 ]; then SPEED=0; fi
                    else
                        SPEED=0
                    fi
                fi

                # Update position based on SPEED and keep inside city bounds (~0.35° box). Heading jitters slightly
                read LAT LON BEARING <<< $(python3 - <<PY
import math,random
base_lat=float("$BASE_LAT"); base_lon=float("$BASE_LON")
lat=float("$LAT"); lon=float("$LON"); speed=float("$SPEED"); dt=float("$STEP_EFFECTIVE")
bearing=float("$BEARING")
R=6378137.0
bearing=(bearing + (random.random()-0.5)*6.0)%360.0
dist_m = max(0.0, speed*1000.0/3600.0*dt)
lat_rad = math.radians(lat)
lat2 = lat + (dist_m/R)*(180.0/math.pi)*math.cos(math.radians(bearing))
lon2 = lon + (dist_m/R)*(180.0/math.pi)*math.sin(math.radians(bearing))/max(1e-6,math.cos(lat_rad))
box=0.35
if not (base_lat-box <= lat2 <= base_lat+box and base_lon-box <= lon2 <= base_lon+box):
    bearing=(bearing+180.0)%360.0
    lat2 = lat + (dist_m/R)*(180.0/math.pi)*math.cos(math.radians(bearing))
    lon2 = lon + (dist_m/R)*(180.0/math.pi)*math.sin(math.radians(bearing))/max(1e-6,math.cos(lat_rad))
print(f"{lat2:.6f} {lon2:.6f} {bearing:.2f}")
PY
)
                if [ "$ENFORCE_SNAP" = "1" ]; then
                    read LAT LON <<< "$(osrm_snap "$LAT" "$LON")"
                fi
                # Energy/fuel consumption and idle refuel/recharge
                if [ "$VEHICLE_TYPE" = "EV" ]; then
                    if [ $SPEED -gt 0 ]; then
                        CONS=$(python3 - <<PY
import random
speed=$SPEED
print(f"{max(0.02, min(0.08, speed/900.0)):.4f}")
PY
)
                        BATTERY_LEVEL=$(python3 - <<PY
lvl=$BATTERY_LEVEL
cons=$CONS
print(f"{max(0.0, lvl - cons):.2f}")
PY
)
                        EV_BATT_UPDATES=$((EV_BATT_UPDATES+1))
                    else
                        if awk "BEGIN{exit !($BATTERY_LEVEL < 35)}"; then
                            if [ $((RANDOM % 100)) -lt 10 ]; then
                                BATTERY_LEVEL=$(python3 - <<PY
lvl=$BATTERY_LEVEL
print(f"{min(100.0, lvl + 0.5):.2f}")
PY
)
                                EV_BATT_UPDATES=$((EV_BATT_UPDATES+1))
                            fi
                        fi
                    fi
                else
                    if [ $SPEED -gt 0 ]; then
                        CONS=$(python3 - <<PY
import random
speed=$SPEED
print(f"{max(0.02, min(0.10, speed/800.0)):.4f}")
PY
)
                        FUEL_LEVEL=$(python3 - <<PY
lvl=$FUEL_LEVEL
cons=$CONS
print(f"{max(0.0, lvl - cons):.2f}")
PY
)
                        ICE_FUEL_UPDATES=$((ICE_FUEL_UPDATES+1))
                    else
                        if awk "BEGIN{exit !($FUEL_LEVEL < 30)}"; then
                            if [ $((RANDOM % 100)) -lt 10 ]; then
                                FUEL_LEVEL=$(python3 - <<PY
lvl=$FUEL_LEVEL
print(f"{min(100.0, lvl + 0.6):.2f}")
PY
)
                                ICE_FUEL_UPDATES=$((ICE_FUEL_UPDATES+1))
                            fi
                        fi
                    fi
                fi

                if [ "$VEHICLE_TYPE" = "EV" ]; then EM=0; else EM=$((SPEED/2)); fi
                ISO_TS=$(date -u -r $ts +%Y-%m-%dT%H:%M:%SZ)
                TELEMETRY_DATA="{\"vehicle_id\": \"$vid\", \"timestamp\": \"$ISO_TS\", \"location\": {\"lat\": $LAT, \"lon\": $LON}, \"speed\": $SPEED, \"fuel_level\": $FUEL_LEVEL, \"battery_level\": $BATTERY_LEVEL, \"emissions\": $EM, \"type\": \"$VEHICLE_TYPE\", \"status\": \"active\" }"
                CODE=$(curl -s -m 5 -o /tmp/tele_body.$$ -w '%{http_code}' -X POST http://localhost:8081/api/telemetry \
                    -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d "$TELEMETRY_DATA")
                if [ "$CODE" -ge 200 ] && [ "$CODE" -lt 300 ]; then
                    TELE_POSTED=$((TELE_POSTED+1))
                    if [ "$VEHICLE_TYPE" = "EV" ]; then EV_POINTS=$((EV_POINTS+1)); else ICE_POINTS=$((ICE_POINTS+1)); fi
                    # Update progress every 50 posts or at least every second
                    if [ $((TELE_POSTED % 50)) -eq 0 ]; then
                        NOW_TS=$(date +%s)
                        if [ $((NOW_TS - LAST_PRINT_TS)) -ge 1 ]; then
                            _telemetry_tick
                            LAST_PRINT_TS=$NOW_TS
                        fi
                    fi
                    # Stop early if we reached cap
                    if [ "$TELE_POSTED" -ge "$TOTAL_POINTS" ]; then
                        break
                    fi
                else
                    # On 401, refresh token once and retry immediately
                    if [ "$CODE" = "401" ]; then
                        get_token
                        RT_CODE=$(curl -s -m 5 -o /tmp/tele_body.$$ -w '%{http_code}' -X POST http://localhost:8081/api/telemetry \
                            -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d "$TELEMETRY_DATA")
                        if [ "$RT_CODE" -ge 200 ] && [ "$RT_CODE" -lt 300 ]; then
                            TELE_POSTED=$((TELE_POSTED+1))
                        fi
                    fi
                    # Sample a few errors for visibility
                    if [ $((TELE_POSTED % 500)) -eq 0 ]; then
                        ERR_SNIP=$(head -c 120 /tmp/tele_body.$$ 2>/dev/null | tr '\n' ' ')
                        print_warning "   Telemetry post failed (HTTP ${CODE}); sample body: ${ERR_SNIP}"
                    fi
                fi
            done
            if [ "$TELE_POSTED" -ge "$TOTAL_POINTS" ]; then
                break
            fi
        done
        progress_done "   Telemetry:" "$TELE_POSTED" "$TOTAL_POINTS"
        print_status "   Telemetry posted: total=$TELE_POSTED (EV=$EV_POINTS, ICE=$ICE_POINTS)"
        print_status "   Updates: EV battery=$EV_BATT_UPDATES, ICE fuel=$ICE_FUEL_UPDATES"
    fi

    if [ "${QUICK:-0}" != "1" ]; then
        # Create dummy trips
        print_status "5. Creating dummy trips..."
        TRIP_TOTAL=$(( ${#VEHICLE_IDS[@]} * 7 ))
        TRIP_CREATED=0
        for vehicle in "${VEHICLE_IDS[@]}"; do
            for day in {1..7}; do
                TRIP_CREATED=$((TRIP_CREATED+1))
                progress_print "   Trips:" "$TRIP_CREATED" "$TRIP_TOTAL"
                # ... existing trip body ...
            RESPONSE=$(curl -s -X POST http://localhost:8081/api/trips \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $TOKEN" \
                -d "$TRIP_DATA")
        done
    done
        progress_done "   Trips:" "$TRIP_CREATED" "$TRIP_TOTAL"
    else
        print_status "5. Skipping trips (Quick Test)"
    fi

    if [ "${QUICK:-0}" != "1" ]; then
    # Create dummy maintenance records
    print_status "6. Creating dummy maintenance records..."
        MAINT_CREATED=0
        MAINT_TOTAL=$(( ${#VEHICLE_IDS[@]} * 3 ))
    for vehicle in "${VEHICLE_IDS[@]}"; do
        NUM_RECORDS=$((RANDOM % 3 + 2))
        for i in $(seq 1 $NUM_RECORDS); do
                MAINT_CREATED=$((MAINT_CREATED+1))
                progress_print "   Maintenance:" "$MAINT_CREATED" "$MAINT_TOTAL"
                # ... existing maintenance body ...
            RESPONSE=$(curl -s -X POST http://localhost:8081/api/maintenance \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $TOKEN" \
                -d "$MAINT_DATA")
        done
    done
        progress_done "   Maintenance:" "$MAINT_CREATED" "$MAINT_TOTAL"
    else
        print_status "6. Skipping maintenance (Quick Test)"
    fi

    if [ "${QUICK:-0}" != "1" ]; then
    # Create dummy cost records
    print_status "7. Creating dummy cost records..."
        COST_CREATED=0
        COST_TOTAL=$(( ${#VEHICLE_IDS[@]} * 4 ))
    for vehicle in "${VEHICLE_IDS[@]}"; do
        NUM_RECORDS=$((RANDOM % 4 + 3))
        for i in $(seq 1 $NUM_RECORDS); do
                COST_CREATED=$((COST_CREATED+1))
                progress_print "   Costs:" "$COST_CREATED" "$COST_TOTAL"
                # ... existing cost body ...
            RESPONSE=$(curl -s -X POST http://localhost:8081/api/costs \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $TOKEN" \
                -d "$COST_DATA")
        done
    done
        progress_done "   Costs:" "$COST_CREATED" "$COST_TOTAL"
    else
        print_status "7. Skipping costs (Quick Test)"
    fi

    echo ""
    print_status "🎉 Database populated successfully!"
    echo ""
    if [ "${QUICK:-0}" = "1" ]; then
        echo -e "${BLUE}Created (Quick Test):${NC}"
        echo "   🚗 1 vehicle (EV)"
        echo "   📊 ~60 telemetry records (last ~10 minutes)"
        if [ -n "${QUICK_VID:-}" ]; then
            echo "   🔎 Vehicle ID: ${QUICK_VID}"
        fi
    else
    echo -e "${BLUE}Created:${NC}"
    echo "   🚗 8 vehicles (ICE and EV with detailed info)"
    echo "   📊 100 telemetry records (last 7 days)"
    echo "   🛣️  Multiple trips per vehicle (last 7 days)"
    echo "   🔧 2-4 maintenance records per vehicle (last 6 months)"
    echo "   💰 3-6 cost records per vehicle (last 3 months)"
    echo ""
    echo -e "${BLUE}Data includes:${NC}"
    echo "   • Realistic vehicle details (VIN, license plates)"
    echo "   • Varied maintenance types and costs"
    echo "   • Different cost categories (fuel, electricity, maintenance, etc.)"
    echo "   • Realistic timestamps and locations"
    echo "   • Multiple technicians and vendors"
    fi
    echo ""
    echo -e "${YELLOW}You can now open Live View to see movement.${NC}"
    echo ""
    
    # If city-based option was used, restart simulator to pick up new cities
    if [ "$psel" = "2" ] && [ -n "${SIM_EXTRA_CITIES:-}" ]; then
        echo ""
        print_status "City-based option detected - restarting simulator with selected cities..."
        
        # Stop existing simulator
        stop_simulator >/dev/null 2>&1 || true
        sleep 2
        
        # For global cities (like India), use public OSRM for better road coverage
        # Local OSRM only has Monaco data, which doesn't cover India well
        print_status "Using public OSRM for global road coverage (includes India)..."
        export OSRM_BASE_URL="https://router.project-osrm.org"
        
        # Use existing vehicles from populate instead of creating new ones
        export SIM_USE_EXISTING=1
        export FLEET_SIZE=8  # Match the number of vehicles created by populate
        
        # Start simulator with the selected cities
        print_status "Starting simulator with cities: ${CITIES[*]}"
        print_status "Environment: SIM_USE_EXISTING=$SIM_USE_EXISTING, FLEET_SIZE=$FLEET_SIZE, SIM_EXTRA_CITIES=$SIM_EXTRA_CITIES"
        start_simulator local
        
        echo ""
        print_status "✅ Simulator restarted with selected cities!"
        print_status "Vehicles should now spawn in and move between your selected cities."
    fi
}

# Function to ensure admin user exists
ensure_admin_user() {
    print_status "Ensuring admin user exists..."
    
    # Wait for backend to be ready
    for i in {1..60}; do
        if curl -s http://localhost:8081 > /dev/null 2>&1; then
            print_status "   Backend is ready"
            break
        fi
        if [ $i -eq 60 ]; then
            print_error "Backend failed to start within 60 seconds"
            return 1
        fi
        sleep 1
    done
    
    # Give backend a moment to fully initialize
    sleep 2
    
    # Try to login first to check if admin exists
    LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8081/api/auth/login \
        -H "Content-Type: application/json" \
        -d '{"username": "admin", "password": "admin123"}')
    
    if echo "$LOGIN_RESPONSE" | grep -q "token"; then
        print_status "   Admin user already exists"
        return 0
    fi
    
    # If we get here, admin doesn't exist or login failed
    print_status "   Admin user not found, creating..."
    
    # Admin doesn't exist, create it
    print_status "   Creating admin user..."
    REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8081/api/auth/register \
        -H "Content-Type: application/json" \
        -d '{"username": "admin", "password": "admin123", "email": "admin@fleet.com", "first_name": "Admin", "last_name": "User", "role": "admin"}')
    
    if echo "$REGISTER_RESPONSE" | grep -q "token\|success"; then
        print_status "   Admin user created successfully"
        return 0
    else
        print_warning "   Failed to create admin user, trying alternative method..."
        
        # Try creating other test users as well
        curl -s -X POST http://localhost:8081/api/auth/register \
            -H "Content-Type: application/json" \
            -d '{"username": "manager", "password": "manager123", "email": "manager@fleet.com", "first_name": "Manager", "last_name": "User", "role": "manager"}' > /dev/null 2>&1
        
        curl -s -X POST http://localhost:8081/api/auth/register \
            -H "Content-Type: application/json" \
            -d '{"username": "operator", "password": "operator123", "email": "operator@fleet.com", "first_name": "Operator", "last_name": "User", "role": "operator"}' > /dev/null 2>&1
        
        curl -s -X POST http://localhost:8081/api/auth/register \
            -H "Content-Type: application/json" \
            -d '{"username": "viewer", "password": "viewer123", "email": "viewer@fleet.com", "first_name": "Viewer", "last_name": "User", "role": "viewer"}' > /dev/null 2>&1
        
        # Try login again
        LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8081/api/auth/login \
            -H "Content-Type: application/json" \
            -d '{"username": "admin", "password": "admin123"}')
        
        if echo "$LOGIN_RESPONSE" | grep -q "token"; then
            print_status "   Admin user is now available"
            return 0
        else
            print_error "   Failed to create admin user"
            print_status "   Response: $REGISTER_RESPONSE"
            return 1
        fi
    fi
}

# Helper: get auth token
get_token() {
    local resp
    resp=$(curl -s -m 5 -X POST http://localhost:8081/api/auth/login \
        -H "Content-Type: application/json" \
        -d '{"username": "admin", "password": "admin123"}') || resp=""
    if command -v jq >/dev/null 2>&1; then
        TOKEN=$(echo "$resp" | jq -r '.token // empty')
    else
        TOKEN=$(echo "$resp" | grep -o '"token":"[^"]*"' | cut -d '"' -f4)
    fi
}

# Simulator controls
start_simulator() {
    print_header
    print_status "Starting simulator (vehicles will move on the map)..."
    echo ""

    # Handle parameters
    MODE="${1:-local}"
    if [ "$MODE" = "global" ]; then
        SIM_GLOBAL=1
        # Force global OSRM unless explicitly overridden by user
        OSRM_URL="${OSRM_BASE_URL:-https://router.project-osrm.org}"
    else
        : "${SIM_GLOBAL:=0}"
        # Respect any existing OSRM_BASE_URL (e.g., local Monaco)
        OSRM_URL="${OSRM_BASE_URL}"
    fi

    # Ensure backend is up
    if ! curl -s http://localhost:8081 > /dev/null 2>&1; then
        print_error "Backend is not running. Run: $0 start"
        return 1
    fi

    # Ensure admin user and get token
    ensure_admin_user
    LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8081/api/auth/login \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}')
    if command -v jq >/dev/null 2>&1; then
        TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token')
    else
        TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d '"' -f4)
    fi
    if [ -z "$TOKEN" ]; then
        print_error "Failed to obtain auth token"
        return 1
    fi

    # Defaults (can be overridden with env vars)
    FLEET_SIZE="${FLEET_SIZE:-1}"
    SIM_TICK_SECONDS="${SIM_TICK_SECONDS:-1}"
    SIM_SNAP_TO_ROAD="${SIM_SNAP_TO_ROAD:-1}"

    # Stop any running simulator first and clean up
    stop_simulator >/dev/null 2>&1 || true
    
    # Additional cleanup - kill any remaining simulator processes
    pkill -f './simulator' >/dev/null 2>&1 || true
    pkill -f 'go run ./cmd/simulator' >/dev/null 2>&1 || true
    
    # Clean up any stale PID files
    rm -f .simulator_pid /tmp/fleet_simulator_pid 2>/dev/null || true

    # Choose binary or go run - ensure we're in the correct project directory
    SCRIPT_DIR="$(dirname "$(realpath "${BASH_SOURCE[0]:-$0}" 2>/dev/null || echo "${BASH_SOURCE[0]:-$0}")")"
    PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Fallback to known project path if path resolution fails
    if [ ! -f "$PROJECT_DIR/go.mod" ]; then
        KNOWN_PROJECT_DIR="/Users/yilmazu/Projects/hephaestus-sytems/Fleet-Sustainability"
        if [ -f "$KNOWN_PROJECT_DIR/go.mod" ]; then
            print_status "Using known project directory: $KNOWN_PROJECT_DIR"
            PROJECT_DIR="$KNOWN_PROJECT_DIR"
        fi
    fi
    
    print_status "Script directory: $SCRIPT_DIR"
    print_status "Project directory: $PROJECT_DIR"
    
    cd "$PROJECT_DIR"
    
    # Ensure we have write permissions in the current directory
    if ! touch test_write_permission 2>/dev/null; then
        print_error "No write permission in current directory: $(pwd)"
        return 1
    fi
    rm -f test_write_permission 2>/dev/null || true
    
    # Check if we're in the right directory (should have go.mod)
    if [ ! -f "go.mod" ]; then
        print_error "go.mod not found in $(pwd). This doesn't appear to be a Go project directory."
        print_status "Looking for go.mod in parent directories..."
        # Try to find go.mod in parent directories
        SEARCH_DIR="$(pwd)"
        while [ "$SEARCH_DIR" != "/" ] && [ ! -f "$SEARCH_DIR/go.mod" ]; do
            SEARCH_DIR="$(dirname "$SEARCH_DIR")"
        done
        if [ -f "$SEARCH_DIR/go.mod" ]; then
            print_status "Found go.mod in $SEARCH_DIR, changing directory..."
            cd "$SEARCH_DIR"
        else
            print_error "Could not find go.mod file. Cannot run Go simulator."
            print_status "Current directory contents:"
            ls -la
            return 1
        fi
    fi
    
    print_status "Working in directory: $(pwd)"
    
    if [ -x "./simulator" ]; then
        SIM_RUN_CMD="./simulator"
        print_status "Using compiled simulator binary"
    else
        # Check if Go is available
        if ! command -v go >/dev/null 2>&1; then
            print_error "Go is not installed or not in PATH"
            return 1
        fi
        # Verify go.mod exists
        if [ ! -f "go.mod" ]; then
            print_error "go.mod file not found in $(pwd)"
            return 1
        fi
        
        # Try to build the simulator first
        print_status "Building simulator binary..."
        if go build -o simulator ./cmd/simulator 2>/dev/null; then
            SIM_RUN_CMD="./simulator"
            print_status "Successfully built simulator binary"
        else
            print_warning "Failed to build simulator binary, using go run instead"
            SIM_RUN_CMD="go run ./cmd/simulator"
            print_status "Using go run ./cmd/simulator"
        fi
    fi

    if [ -n "$OSRM_URL" ]; then
        print_status "Using OSRM at $OSRM_URL (MODE=$MODE)"
    else
        print_warning "OSRM_BASE_URL not set; simulator will default to public OSRM internally"
    fi

    # Create/clear output files with proper permissions
    rm -f simulator.out .simulator_pid 2>/dev/null || true
    touch simulator.out .simulator_pid
    chmod 666 simulator.out .simulator_pid
    
    # Launch in background with robust file handling
    if [ -n "$OSRM_URL" ]; then
        {
            env \
                SIM_AUTH_TOKEN="$TOKEN" \
                API_BASE_URL="http://localhost:8081/api" \
                SIM_TICK_SECONDS="$SIM_TICK_SECONDS" \
                FLEET_SIZE="$FLEET_SIZE" \
                SIM_SNAP_TO_ROAD="$SIM_SNAP_TO_ROAD" \
                SIM_GLOBAL="$SIM_GLOBAL" \
                SIM_USE_EXISTING="${SIM_USE_EXISTING:-0}" \
                SIM_EXTRA_CITIES="${SIM_EXTRA_CITIES:-}" \
                OSRM_BASE_URL="$OSRM_URL" \
                $SIM_RUN_CMD
        } > simulator.out 2>&1 &
    else
        {
            env \
                SIM_AUTH_TOKEN="$TOKEN" \
                API_BASE_URL="http://localhost:8081/api" \
                SIM_TICK_SECONDS="$SIM_TICK_SECONDS" \
                FLEET_SIZE="$FLEET_SIZE" \
                SIM_SNAP_TO_ROAD="$SIM_SNAP_TO_ROAD" \
                SIM_GLOBAL="$SIM_GLOBAL" \
                SIM_USE_EXISTING="${SIM_USE_EXISTING:-0}" \
                SIM_EXTRA_CITIES="${SIM_EXTRA_CITIES:-}" \
                $SIM_RUN_CMD
        } > simulator.out 2>&1 &
    fi
    SIM_PID=$!
    sleep 0.5  # Longer delay to ensure process has started
    
    # Verify the process is actually running
    if ! kill -0 "$SIM_PID" >/dev/null 2>&1; then
        print_error "Simulator process failed to start (PID: $SIM_PID)"
        print_status "Checking simulator.out for error details:"
        if [ -f "simulator.out" ]; then
            tail -10 simulator.out
        else
            print_warning "No simulator.out file found"
        fi
        
        # Try auto-fix as a last resort
        print_status "Attempting auto-fix to resolve issues..."
        auto_fix >/dev/null 2>&1 || true
        
        # Try starting simulator again after auto-fix
        print_status "Retrying simulator start after auto-fix..."
        sleep 2
        
        # Re-run the simulator start process
        if [ -n "$OSRM_URL" ]; then
            {
                env \
                    SIM_AUTH_TOKEN="$TOKEN" \
                    API_BASE_URL="http://localhost:8081/api" \
                    SIM_TICK_SECONDS="$SIM_TICK_SECONDS" \
                    FLEET_SIZE="$FLEET_SIZE" \
                    SIM_SNAP_TO_ROAD="$SIM_SNAP_TO_ROAD" \
                    SIM_GLOBAL="$SIM_GLOBAL" \
                    SIM_USE_EXISTING="${SIM_USE_EXISTING:-0}" \
                    SIM_EXTRA_CITIES="${SIM_EXTRA_CITIES:-}" \
                    OSRM_BASE_URL="$OSRM_URL" \
                    $SIM_RUN_CMD
            } > simulator.out 2>&1 &
        else
            {
                env \
                    SIM_AUTH_TOKEN="$TOKEN" \
                    API_BASE_URL="http://localhost:8081/api" \
                    SIM_TICK_SECONDS="$SIM_TICK_SECONDS" \
                    FLEET_SIZE="$FLEET_SIZE" \
                    SIM_SNAP_TO_ROAD="$SIM_SNAP_TO_ROAD" \
                    SIM_GLOBAL="$SIM_GLOBAL" \
                    SIM_USE_EXISTING="${SIM_USE_EXISTING:-0}" \
                    SIM_EXTRA_CITIES="${SIM_EXTRA_CITIES:-}" \
                    $SIM_RUN_CMD
            } > simulator.out 2>&1 &
        fi
        
        SIM_PID=$!
        sleep 1
        
        # Check if retry worked
        if kill -0 "$SIM_PID" >/dev/null 2>&1; then
            print_status "Simulator started successfully on retry (PID: $SIM_PID)"
        else
            print_error "Simulator still failed to start after auto-fix"
            return 1
        fi
    fi
    
    # Write PID file with robust error handling
    {
        printf "%s\n" "$SIM_PID" > .simulator_pid 2>/dev/null && print_status "PID file written successfully" || {
            print_warning "Could not write to .simulator_pid file - trying alternative method"
            # Try alternative methods
            printf "%s\n" "$SIM_PID" > .simulator_pid 2>/dev/null || {
                # Use a different approach
                echo "$SIM_PID" | tee .simulator_pid >/dev/null 2>&1 || {
                    print_warning "All methods failed to write PID file - using backup location"
                    # Create a backup method
                    printf "%s\n" "$SIM_PID" > /tmp/fleet_simulator_pid 2>/dev/null || true
                }
            }
        }
    }
    
    # Verify PID file was created
    if [ -f ".simulator_pid" ] || [ -f "/tmp/fleet_simulator_pid" ]; then
        print_status "Simulator started successfully (PID: $SIM_PID). Logs: simulator.out"
    else
        print_warning "Simulator started (PID: $SIM_PID) but PID file creation failed"
    fi
}

stop_simulator() {
    print_header
    print_status "Stopping simulator..."
    cd "$(dirname "$(realpath "${BASH_SOURCE[0]:-$0}" 2>/dev/null || echo "${BASH_SOURCE[0]:-$0}")")/.."
    if [ -f ".simulator_pid" ]; then
        SIM_PID=$(cat .simulator_pid)
        if kill -0 "$SIM_PID" >/dev/null 2>&1; then
            kill "$SIM_PID" >/dev/null 2>&1 || true
            sleep 1
        fi
        rm -f .simulator_pid
    elif [ -f "/tmp/fleet_simulator_pid" ]; then
        SIM_PID=$(cat /tmp/fleet_simulator_pid)
        if kill -0 "$SIM_PID" >/dev/null 2>&1; then
            kill "$SIM_PID" >/dev/null 2>&1 || true
            sleep 1
        fi
        rm -f /tmp/fleet_simulator_pid
    fi
    # Fallback
    pkill -f './simulator' >/dev/null 2>&1 || pkill -f 'go run ./cmd/simulator' >/dev/null 2>&1 || true
    print_status "Simulator stopped (if it was running)."
}

simulator_status() {
    print_header
    print_status "Simulator status:"
    cd "$(dirname "$(realpath "${BASH_SOURCE[0]:-$0}" 2>/dev/null || echo "${BASH_SOURCE[0]:-$0}")")/.."
    
    # Check primary PID file
    if [ -f ".simulator_pid" ]; then
        SIM_PID=$(cat .simulator_pid)
        if kill -0 "$SIM_PID" >/dev/null 2>&1; then
            echo "PID: $SIM_PID (running)"
            return 0
        else
            echo "PID file present but process not running (PID: $SIM_PID)"
        fi
    fi
    
    # Check backup PID file
    if [ -f "/tmp/fleet_simulator_pid" ]; then
        SIM_PID=$(cat /tmp/fleet_simulator_pid)
        if kill -0 "$SIM_PID" >/dev/null 2>&1; then
            echo "PID: $SIM_PID (running) - using backup PID file"
            return 0
        else
            echo "Backup PID file present but process not running (PID: $SIM_PID)"
        fi
    fi
    
    # Try to detect by process name
    SIM_PID=$(pgrep -f './simulator' 2>/dev/null | head -1)
    if [ -n "$SIM_PID" ]; then
        echo "Running (process detected: PID $SIM_PID), but no PID file"
        return 0
    fi
    
    SIM_PID=$(pgrep -f 'go run ./cmd/simulator' 2>/dev/null | head -1)
    if [ -n "$SIM_PID" ]; then
        echo "Running (go run process detected: PID $SIM_PID), but no PID file"
        return 0
    fi
    
    echo "Not running"
}

# OSRM (routing) controls
check_osrm() {
    # Determine base URL (public by default)
    OSRM_BASE_URL=${OSRM_BASE_URL:-https://router.project-osrm.org}
    TEST_URL="$OSRM_BASE_URL/route/v1/driving/0,0;0.1,0.1?overview=false"
    if command -v curl >/dev/null 2>&1; then
        CODE=$(curl -s -o /dev/null -w '%{http_code}' "$TEST_URL" || echo "000")
    else
        CODE="000"
    fi
    if [ "$CODE" = "200" ]; then
        print_status "✅ OSRM reachable at $OSRM_BASE_URL"
        return 0
    else
        print_warning "❌ OSRM not reachable at $OSRM_BASE_URL (HTTP $CODE)"
        echo "   Vehicles will move but may not follow roads without OSRM."
        return 1
    fi
}

start_local_osrm() {
    print_status "Starting local OSRM (Monaco dataset) on http://localhost:5000 ..."
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker is not running; cannot start local OSRM."
        return 1
    fi
    cd "$(dirname "$(realpath "${BASH_SOURCE[0]:-$0}" 2>/dev/null || echo "${BASH_SOURCE[0]:-$0}")")/.."
    OSRM_DIR="$(pwd)/build/osrm"
    mkdir -p "$OSRM_DIR"
    DATA_PBF="$OSRM_DIR/monaco-latest.osm.pbf"
    if [ ! -f "$DATA_PBF" ]; then
        print_status "Downloading sample dataset (Monaco) ..."
        curl -L -o "$DATA_PBF" https://download.geofabrik.de/europe/monaco-latest.osm.pbf || {
            print_error "Failed to download Monaco dataset."; return 1; }
    fi
    # Prepare data (extract/partition/customize) if not prepared
    if [ ! -f "$OSRM_DIR/monaco-latest.osrm" ]; then
        print_status "Preparing OSRM graph (this may take a minute) ..."
        docker run --rm -t -v "$OSRM_DIR":/data osrm/osrm-backend osrm-extract -p /opt/car.lua /data/monaco-latest.osm.pbf || return 1
        docker run --rm -t -v "$OSRM_DIR":/data osrm/osrm-backend osrm-partition /data/monaco-latest.osrm || return 1
        docker run --rm -t -v "$OSRM_DIR":/data osrm/osrm-backend osrm-customize /data/monaco-latest.osrm || return 1
    fi
    # Start routed if not running
    if docker ps --format '{{.Names}}' | grep -q '^fleet-osrm$'; then
        print_status "Local OSRM already running"
    else
        docker rm -f fleet-osrm >/dev/null 2>&1 || true
        docker run -d --name fleet-osrm -p 5000:5000 -v "$OSRM_DIR":/data osrm/osrm-backend osrm-routed --algorithm mld /data/monaco-latest.osrm >/dev/null || {
            print_error "Failed to start OSRM container."; return 1; }
        # Wait until ready
        for i in {1..20}; do
            CODE=$(curl -s -o /dev/null -w '%{http_code}' 'http://localhost:5000/route/v1/driving/7.41,43.73;7.42,43.74?overview=false' || echo 000)
            [ "$CODE" = "200" ] && break
            sleep 1
        done
        if [ "$CODE" != "200" ]; then
            print_warning "OSRM container started but not responding yet."
        fi
    fi
    export OSRM_BASE_URL="http://localhost:5000"
    print_status "Local OSRM ready at $OSRM_BASE_URL"
}

stop_local_osrm() {
    print_status "Stopping local OSRM ..."
    docker rm -f fleet-osrm >/dev/null 2>&1 || true
    print_status "Local OSRM stopped."
}

osrm_status() {
    print_status "Checking OSRM status ..."
    if docker ps --format '{{.Names}}' | grep -q '^fleet-osrm$'; then
        echo "Local OSRM container: running (fleet-osrm)"
    else
        echo "Local OSRM container: not running"
    fi
    check_osrm >/dev/null 2>&1 || true
}

# Auto-fix flow: reset DB, seed few vehicles, start simulator using existing vehicles, verify telemetry
auto_fix() {
    print_header
    print_status "Running Auto-fix: stop sim, clear DB, seed, start movement..."

    # 1) Stop simulator
    stop_simulator >/dev/null 2>&1 || true

    # 2) Clear DB (preserves users)
    clear_database || { print_error "Auto-fix aborted: failed to clear DB"; return 1; }

    # 3) Ensure backend and admin token
    print_status "Ensuring backend and admin token..."
    ensure_admin_user || { print_error "Auto-fix aborted: backend/admin not ready"; return 1; }
    LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8081/api/auth/login \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}')
    if command -v jq >/dev/null 2>&1; then
        TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token // empty')
    else
        TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d '"' -f4)
    fi
    if [ -z "$TOKEN" ]; then
        print_error "Failed to obtain admin token"
        return 1
    fi

    # Ensure OSRM available for snapping-to-road
    print_status "Checking routing (OSRM) for snapping-to-road..."
    OSRM_EFF_URL=${OSRM_BASE_URL:-https://router.project-osrm.org}
    CODE=$(curl -s -o /dev/null -w '%{http_code}' "$OSRM_EFF_URL/route/v1/driving/0,0;0.1,0.1?overview=false" || echo 000)
    if [ "$CODE" != "200" ]; then
        print_warning "Public OSRM not reachable (HTTP $CODE); attempting local OSRM..."
        start_local_osrm || true
        OSRM_EFF_URL="http://localhost:5000"
        for i in {1..20}; do
            CODE=$(curl -s -o /dev/null -w '%{http_code}' "$OSRM_EFF_URL/route/v1/driving/7.41,43.73;7.42,43.74?overview=false" || echo 000)
            [ "$CODE" = "200" ] && break
            sleep 1
        done
    fi
    if [ "$CODE" = "200" ]; then
        export OSRM_BASE_URL="$OSRM_EFF_URL"
        ENFORCE_SNAP=1
        print_status "OSRM reachable at $OSRM_EFF_URL (snapping enforced)"
    else
        ENFORCE_SNAP=0
        print_warning "OSRM not reachable; seed may not snap precisely to roads."
    fi

    # 4) Seed vehicles: exactly 1 ICE and 1 EV
    print_status "Seeding vehicles (1 ICE, 1 EV)..."
    
    # Use NYC area for both vehicles
    BASE_LAT="40.7128"
    BASE_LON="-74.0060"
    
    VEH_CREATED=0
    
    # Create 1 ICE vehicle
    read V_LAT V_LON <<< $(python3 - <<PY
import math,random
base_lat=float("$BASE_LAT"); base_lon=float("$BASE_LON")
R=6378137.0
radius_m=1500.0*random.random()
theta=2*math.pi*random.random()
dlat=(radius_m/R)*(180.0/math.pi)
dlon=(radius_m/(R*max(1e-6,math.cos(math.radians(base_lat)))))*(180.0/math.pi)
lat=base_lat + dlat*math.cos(theta)
lon=base_lon + dlon*math.sin(theta)
print(f"{lat:.6f} {lon:.6f}")
PY
)
    VEHICLE_BODY=$(cat <<JSON
{"type":"ICE","make":"Toyota","model":"Camry","year":2023,"status":"active","current_location":{"lat":$V_LAT,"lon":$V_LON}}
JSON
)
    CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8081/api/vehicles \
        -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" \
        -d "$VEHICLE_BODY")
    if [ "$CODE" -ge 200 ] && [ "$CODE" -lt 300 ]; then
        VEH_CREATED=$((VEH_CREATED+1))
        print_status "ICE vehicle created"
    fi
    
    # Create 1 EV vehicle
    read V_LAT V_LON <<< $(python3 - <<PY
import math,random
base_lat=float("$BASE_LAT"); base_lon=float("$BASE_LON")
R=6378137.0
radius_m=1500.0*random.random()
theta=2*math.pi*random.random()
dlat=(radius_m/R)*(180.0/math.pi)
dlon=(radius_m/(R*max(1e-6,math.cos(math.radians(base_lat)))))*(180.0/math.pi)
lat=base_lat + dlat*math.cos(theta)
lon=base_lon + dlon*math.sin(theta)
print(f"{lat:.6f} {lon:.6f}")
PY
)
    VEHICLE_BODY=$(cat <<JSON
{"type":"EV","make":"Tesla","model":"Model 3","year":2023,"status":"active","current_location":{"lat":$V_LAT,"lon":$V_LON}}
JSON
)
    CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8081/api/vehicles \
        -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" \
        -d "$VEHICLE_BODY")
    if [ "$CODE" -ge 200 ] && [ "$CODE" -lt 300 ]; then
        VEH_CREATED=$((VEH_CREATED+1))
        print_status "EV vehicle created"
    fi
    
    print_status "Vehicles created: $VEH_CREATED (1 ICE, 1 EV)"

    # 4b) Seed commute-style telemetry to make graphs immediately useful
    print_status "Seeding commute-style telemetry for graphs..."
    VEHICLE_IDS=()
    VEHICLE_TYPES=()
    VEHICLES_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/vehicles)
    if command -v jq >/dev/null 2>&1; then
        while read -r id type; do
            [ -n "$id" ] && VEHICLE_IDS+=("$id") && VEHICLE_TYPES+=("$type")
        done < <(echo "$VEHICLES_RESPONSE" | jq -r '.[]? | "\(.id // empty) \(.type // empty)"')
        if [ ${#VEHICLE_IDS[@]} -eq 0 ]; then
            while read -r id type; do [ -n "$id" ] && VEHICLE_IDS+=("$id") && VEHICLE_TYPES+=("$type"); done < <(echo "$VEHICLES_RESPONSE" | jq -r '.data[]? | "\(.id // empty) \(.type // empty)"')
        fi
    fi
    if [ ${#VEHICLE_IDS[@]} -eq 0 ]; then 
        # Fallback: extract both ID and type using grep/sed
        while IFS= read -r line; do
            id=$(echo "$line" | grep -o '"id":"[a-f0-9]\{24\}"' | cut -d '"' -f4)
            type=$(echo "$line" | grep -o '"type":"[^"]*"' | cut -d '"' -f4)
            [ -n "$id" ] && VEHICLE_IDS+=("$id") && VEHICLE_TYPES+=("$type")
        done < <(echo "$VEHICLES_RESPONSE" | grep -E '"id":"[a-f0-9]{24}"' | head -10)
    fi
    NOW_EPOCH=$(date -u +%s)
    START_EPOCH=$(( NOW_EPOCH - 1800 ))
    STEP_SECONDS=30
    VEH_COUNT=${#VEHICLE_IDS[@]}
    RAW_PER_VEH=$(( (NOW_EPOCH-START_EPOCH) / STEP_SECONDS ))
    if [ "$RAW_PER_VEH" -lt 1 ]; then RAW_PER_VEH=1; fi
    TOTAL_RAW=$(( RAW_PER_VEH * VEH_COUNT ))
    MAX_POINTS=${MAX_COMMUTE_POINTS:-900}
    STRIDE=1
    STEP_EFFECTIVE=$STEP_SECONDS
    TOTAL_POINTS=$TOTAL_RAW
    if [ "$TOTAL_RAW" -gt "$MAX_POINTS" ]; then
        STRIDE=$(( (TOTAL_RAW + MAX_POINTS - 1) / MAX_POINTS ))
        if [ "$STRIDE" -lt 1 ]; then STRIDE=1; fi
        STEP_EFFECTIVE=$(( STEP_SECONDS * STRIDE ))
        TOTAL_POINTS=$MAX_POINTS
    fi
    # Adjust denominator to actual planned iterations after stride so progress reaches 100%
    PLANNED_PER_VEH=$(( ((NOW_EPOCH-START_EPOCH) / STEP_EFFECTIVE) + 1 ))
    PLANNED_TOTAL=$(( PLANNED_PER_VEH * VEH_COUNT ))
    if [ "$TOTAL_POINTS" -gt "$PLANNED_TOTAL" ]; then TOTAL_POINTS=$PLANNED_TOTAL; fi
    TELE_POSTED=0
    TELE_ATTEMPTED=0
    progress_print "   Commute telemetry:" "$TELE_ATTEMPTED" "$TOTAL_POINTS"
    LAST_PRINT_TS=$(date +%s)
    for idx in "${!VEHICLE_IDS[@]}"; do
        vid=${VEHICLE_IDS[$idx]}
        vtype=${VEHICLE_TYPES[$idx]}
        CITY_INDEX=$((RANDOM % ${#CITIES[@]}))
        CITY_COORDS="${CITIES[$CITY_INDEX]}"
        BASE_LAT=$(echo "$CITY_COORDS" | cut -d':' -f1)
        BASE_LON=$(echo "$CITY_COORDS" | cut -d':' -f2)
        read LAT LON <<< $(python3 - <<PY
import math,random
base_lat=float("$BASE_LAT"); base_lon=float("$BASE_LON")
R=6378137.0
radius_m=2000.0*random.random()
theta=2*math.pi*random.random()

dlat=(radius_m/R)*(180.0/math.pi)

dlon=(radius_m/(R*max(1e-6,math.cos(math.radians(base_lat)))))*(180.0/math.pi)

lat=base_lat + dlat*math.cos(theta)

lon=base_lon + dlon*math.sin(theta)

print(f"{lat:.6f} {lon:.6f}")
PY
)
        # Snap initial point to road if OSRM is available
        if [ "$ENFORCE_SNAP" = "1" ]; then
            read LAT LON <<< "$(osrm_snap "$LAT" "$LON")"
        fi
        BEARING=$(python3 - <<PY
import random
print(f"{random.random()*360:.2f}")
PY
)
        LOCALK=${COMM_LOCAL_KMH:-10}
        HWK=${COMM_HIGHWAY_KMH:-50}
        LOCAL_HOLD=${COMM_LOCAL_HOLD_SECS:-120}
        ACCEL=${COMM_ACCEL_SECS:-60}
        HW_HOLD=${COMM_HIGHWAY_HOLD_SECS:-240}
        DECEL=${COMM_DECEL_SECS:-60}
        LOCAL_HOLD2=${COMM_LOCAL_HOLD_SECS:-120}
        PARK=${COMM_PARK_SECS:-180}
        CYCLE=$((LOCAL_HOLD + ACCEL + HW_HOLD + DECEL + LOCAL_HOLD2 + PARK))
        if [ -z "$vtype" ]; then VEHICLE_TYPE=$( [ $((RANDOM%2)) -eq 0 ] && echo EV || echo ICE ); else VEHICLE_TYPE=$vtype; fi
        FUEL_LEVEL=95.0
        BATTERY_LEVEL=95.0
        for ((ts=$START_EPOCH; ts<=$NOW_EPOCH; ts+=$STEP_EFFECTIVE)); do
            ELAPSED=$((ts-START_EPOCH))
            TMOD=$((ELAPSED % CYCLE))
            if [ $TMOD -lt $LOCAL_HOLD ]; then
                SPEED=$LOCALK
            elif [ $TMOD -lt $((LOCAL_HOLD + ACCEL)) ]; then
                T=$((TMOD - LOCAL_HOLD))
                SPEED=$(python3 - <<PY
loc=$LOCALK
hw=$HWK
acc=$ACCEL
t=$T
print(int(loc + (hw-loc)*t/acc))
PY
)
            elif [ $TMOD -lt $((LOCAL_HOLD + ACCEL + HW_HOLD)) ]; then
                SPEED=$HWK
            elif [ $TMOD -lt $((LOCAL_HOLD + ACCEL + HW_HOLD + DECEL)) ]; then
                T=$((TMOD - LOCAL_HOLD - ACCEL - HW_HOLD))
                SPEED=$(python3 - <<PY
loc=$LOCALK
hw=$HWK
dec=$DECEL
t=$T
print(int(hw - (hw-loc)*t/dec))
PY
)
            elif [ $TMOD -lt $((LOCAL_HOLD + ACCEL + HW_HOLD + DECEL + LOCAL_HOLD2)) ]; then
                SPEED=$LOCALK
            else
                SPEED=0
            fi
            read LAT LON BEARING <<< $(python3 - <<PY
import math,random
base_lat=float("$BASE_LAT"); base_lon=float("$BASE_LON")
lat=float("$LAT"); lon=float("$LON"); speed=float("$SPEED"); dt=float("$STEP_EFFECTIVE")
bearing=float("$BEARING")
R=6378137.0
bearing=(bearing + (random.random()-0.5)*6.0)%360.0
dist_m = max(0.0, speed*1000.0/3600.0*dt)
lat_rad = math.radians(lat)
lat2 = lat + (dist_m/R)*(180.0/math.pi)*math.cos(math.radians(bearing))
lon2 = lon + (dist_m/R)*(180.0/math.pi)*math.sin(math.radians(bearing))/max(1e-6,math.cos(lat_rad))
box=0.35
if not (base_lat-box <= lat2 <= base_lat+box and base_lon-box <= lon2 <= base_lon+box):
    bearing=(bearing+180.0)%360.0
    lat2 = lat + (dist_m/R)*(180.0/math.pi)*math.cos(math.radians(bearing))
    lon2 = lon + (dist_m/R)*(180.0/math.pi)*math.sin(math.radians(bearing))/max(1e-6,math.cos(lat_rad))
print(f"{lat2:.6f} {lon2:.6f} {bearing:.2f}")
PY
)
            if [ "$ENFORCE_SNAP" = "1" ]; then
                read LAT LON <<< "$(osrm_snap "$LAT" "$LON")"
            fi
            # Snap evolving point to road if OSRM is available
            if [ "$ENFORCE_SNAP" = "1" ]; then
                read LAT LON <<< "$(osrm_snap "$LAT" "$LON")"
            fi
            # Energy/fuel consumption and idle refuel/recharge
            if [ "$VEHICLE_TYPE" = "EV" ]; then
                if [ $SPEED -gt 0 ]; then
                    CONS=$(python3 - <<PY
import random
speed=$SPEED
print(f"{max(0.02, min(0.08, speed/900.0)):.4f}")
PY
)
                    BATTERY_LEVEL=$(python3 - <<PY
lvl=$BATTERY_LEVEL
cons=$CONS
print(f"{max(0.0, lvl - cons):.2f}")
PY
)
                else
                    if awk "BEGIN{exit !($BATTERY_LEVEL < 35)}"; then
                        if [ $((RANDOM % 100)) -lt 10 ]; then
                            BATTERY_LEVEL=$(python3 - <<PY
lvl=$BATTERY_LEVEL
print(f"{min(100.0, lvl + 0.5):.2f}")
PY
)
                        fi
                    fi
                fi
            else
                if [ $SPEED -gt 0 ]; then
                    CONS=$(python3 - <<PY
import random
speed=$SPEED
print(f"{max(0.02, min(0.10, speed/800.0)):.4f}")
PY
)
                    FUEL_LEVEL=$(python3 - <<PY
lvl=$FUEL_LEVEL
cons=$CONS
print(f"{max(0.0, lvl - cons):.2f}")
PY
)
                else
                    if awk "BEGIN{exit !($FUEL_LEVEL < 30)}"; then
                        if [ $((RANDOM % 100)) -lt 10 ]; then
                            FUEL_LEVEL=$(python3 - <<PY
lvl=$FUEL_LEVEL
print(f"{min(100.0, lvl + 0.6):.2f}")
PY
)
                        fi
                    fi
                fi
            fi
            if [ "$VEHICLE_TYPE" = "EV" ]; then EM=0; else EM=$((SPEED/2)); fi
            ISO_TS=$(date -u -r $ts +%Y-%m-%dT%H:%M:%SZ)
            TELEMETRY_DATA="{\"vehicle_id\": \"$vid\", \"timestamp\": \"$ISO_TS\", \"location\": {\"lat\": $LAT, \"lon\": $LON}, \"speed\": $SPEED, \"fuel_level\": $FUEL_LEVEL, \"battery_level\": $BATTERY_LEVEL, \"emissions\": $EM, \"type\": \"$VEHICLE_TYPE\", \"status\": \"active\" }"
            CODE=$(curl -s -m 5 -o /dev/null -w '%{http_code}' -X POST http://localhost:8081/api/telemetry \
                -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d "$TELEMETRY_DATA")
            TELE_ATTEMPTED=$((TELE_ATTEMPTED+1))
            if [ "$CODE" -ge 200 ] && [ "$CODE" -lt 300 ]; then
                TELE_POSTED=$((TELE_POSTED+1))
            fi
            if [ $((TELE_ATTEMPTED % 20)) -eq 0 ]; then
                NOW_TS=$(date +%s)
                if [ $((NOW_TS - LAST_PRINT_TS)) -ge 1 ]; then
                    progress_print "   Commute telemetry:" "$TELE_ATTEMPTED" "$TOTAL_POINTS"
                    LAST_PRINT_TS=$NOW_TS
                fi
            fi
            if [ "$TELE_ATTEMPTED" -ge "$TOTAL_POINTS" ]; then
                break
            fi
        done
        if [ "$TELE_ATTEMPTED" -ge "$TOTAL_POINTS" ]; then
            break
        fi
    done
    progress_done "   Commute telemetry:" "$TELE_ATTEMPTED" "$TOTAL_POINTS"
    print_status "   Commute posted: $TELE_POSTED accepted / $TELE_ATTEMPTED attempted"

    # 4c) Seed historical telemetry for 24h, 7d, 30d (downsampled for performance)
    print_status "Seeding historical telemetry (24h/7d/30d)..."
    for WINDOW in 24h 7d 30d; do
        case "$WINDOW" in
            24h) DUR=$((24*3600)); STEP=${HIST_24H_STEP_SECS:-600}; MAX=${HIST_24H_MAX_POINTS:-1500} ;;
            7d)  DUR=$((7*24*3600)); STEP=${HIST_7D_STEP_SECS:-3600}; MAX=${HIST_7D_MAX_POINTS:-2000} ;;
            30d) DUR=$((30*24*3600)); STEP=${HIST_30D_STEP_SECS:-14400}; MAX=${HIST_30D_MAX_POINTS:-2500} ;;
        esac
        NOW_EPOCH=$(date -u +%s)
        START_EPOCH=$(( NOW_EPOCH - DUR ))
        RAW_PER_VEH=$(( DUR / STEP ))
        if [ "$RAW_PER_VEH" -lt 1 ]; then RAW_PER_VEH=1; fi
        TOTAL_RAW=$(( RAW_PER_VEH * ${#VEHICLE_IDS[@]} ))
        STRIDE=1
        STEP_EFFECTIVE=$STEP
        TOTAL_POINTS=$TOTAL_RAW
        if [ "$TOTAL_RAW" -gt "$MAX" ]; then
            STRIDE=$(( (TOTAL_RAW + MAX - 1) / MAX ))
            if [ "$STRIDE" -lt 1 ]; then STRIDE=1; fi
            STEP_EFFECTIVE=$(( STEP * STRIDE ))
            TOTAL_POINTS=$MAX
        fi
        COUNT=0
        ATTEMPTED=0
        progress_print "   ${WINDOW} telemetry:" "$ATTEMPTED" "$TOTAL_POINTS"
        for idx in "${!VEHICLE_IDS[@]}"; do
            vid=${VEHICLE_IDS[$idx]}; vtype=${VEHICLE_TYPES[$idx]}
            CITY_INDEX=$((RANDOM % ${#CITIES[@]}))
            CITY_COORDS="${CITIES[$CITY_INDEX]}"
            BASE_LAT=$(echo "$CITY_COORDS" | cut -d':' -f1)
            BASE_LON=$(echo "$CITY_COORDS" | cut -d':' -f2)
            read LAT LON <<< $(python3 - <<PY
import math,random
base_lat=float("$BASE_LAT"); base_lon=float("$BASE_LON")
R=6378137.0
radius_m=2000.0*random.random()
theta=2*math.pi*random.random()
dlat=(radius_m/R)*(180.0/math.pi)
dlon=(radius_m/(R*max(1e-6,math.cos(math.radians(base_lat)))))*(180.0/math.pi)
lat=base_lat + dlat*math.cos(theta)
lon=base_lon + dlon*math.sin(theta)
print(f"{lat:.6f} {lon:.6f}")
PY
)
            BEARING=$(python3 - <<PY
import random
print(f"{random.random()*360:.2f}")
PY
)
            if [ -z "$vtype" ]; then VEHICLE_TYPE=$( [ $((RANDOM%2)) -eq 0 ] && echo EV || echo ICE ); else VEHICLE_TYPE=$vtype; fi
            FUEL_LEVEL=$((RANDOM % 41 + 60))
            BATTERY_LEVEL=$((RANDOM % 41 + 60))
            LOCALK=${COMM_LOCAL_KMH:-10}; HWK=${COMM_HIGHWAY_KMH:-50}
            LOCAL_HOLD=${COMM_LOCAL_HOLD_SECS:-180}; ACCEL=${COMM_ACCEL_SECS:-90}
            HW_HOLD=${COMM_HIGHWAY_HOLD_SECS:-300}; DECEL=${COMM_DECEL_SECS:-90}
            LOCAL_HOLD2=${COMM_LOCAL_HOLD_SECS:-180}; PARK=${COMM_PARK_SECS:-600}
            CYCLE=$((LOCAL_HOLD + ACCEL + HW_HOLD + DECEL + LOCAL_HOLD2 + PARK))
            for ((ts=$START_EPOCH; ts<=$NOW_EPOCH; ts+=$STEP_EFFECTIVE)); do
                ELAPSED=$((ts-START_EPOCH))
                TMOD=$((ELAPSED % CYCLE))
                if [ $TMOD -lt $LOCAL_HOLD ]; then
                    SPEED=$LOCALK
                elif [ $TMOD -lt $((LOCAL_HOLD + ACCEL)) ]; then
                    T=$((TMOD - LOCAL_HOLD))
                    SPEED=$(python3 - <<PY
loc=$LOCALK
hw=$HWK
acc=$ACCEL
t=$T
print(int(loc + (hw-loc)*t/acc))
PY
)
                elif [ $TMOD -lt $((LOCAL_HOLD + ACCEL + HW_HOLD)) ]; then
                    SPEED=$HWK
                elif [ $TMOD -lt $((LOCAL_HOLD + ACCEL + HW_HOLD + DECEL)) ]; then
                    T=$((TMOD - LOCAL_HOLD - ACCEL - HW_HOLD))
                    SPEED=$(python3 - <<PY
loc=$LOCALK
hw=$HWK
dec=$DECEL
t=$T
print(int(hw - (hw-loc)*t/dec))
PY
)
                elif [ $TMOD -lt $((LOCAL_HOLD + ACCEL + HW_HOLD + DECEL + LOCAL_HOLD2)) ]; then
                    SPEED=$LOCALK
                else
                    SPEED=0
                fi
                read LAT LON BEARING <<< $(python3 - <<PY
import math,random
base_lat=float("$BASE_LAT"); base_lon=float("$BASE_LON")
lat=float("$LAT"); lon=float("$LON"); speed=float("$SPEED"); dt=float("$STEP_EFFECTIVE")
bearing=float("$BEARING")
R=6378137.0
bearing=(bearing + (random.random()-0.5)*6.0)%360.0
dist_m = max(0.0, speed*1000.0/3600.0*dt)
lat_rad = math.radians(lat)
lat2 = lat + (dist_m/R)*(180.0/math.pi)*math.cos(math.radians(bearing))
lon2 = lon + (dist_m/R)*(180.0/math.pi)*math.sin(math.radians(bearing))/max(1e-6,math.cos(lat_rad))
box=0.40
if not (base_lat-box <= lat2 <= base_lat+box and base_lon-box <= lon2 <= base_lon+box):
    bearing=(bearing+180.0)%360.0
    lat2 = lat + (dist_m/R)*(180.0/math.pi)*math.cos(math.radians(bearing))
    lon2 = lon + (dist_m/R)*(180.0/math.pi)*math.sin(math.radians(bearing))/max(1e-6,math.cos(lat_rad))
print(f"{lat2:.6f} {lon2:.6f} {bearing:.2f}")
PY
)
                # Energy/fuel consumption and idle refuel/recharge
                if [ "$VEHICLE_TYPE" = "EV" ]; then
                    if [ $SPEED -gt 0 ]; then
                        CONS=$(python3 - <<PY
import random
speed=$SPEED
print(f"{max(0.02, min(0.08, speed/900.0)):.4f}")
PY
)
                        BATTERY_LEVEL=$(python3 - <<PY
lvl=$BATTERY_LEVEL
cons=$CONS
print(f"{max(0.0, lvl - cons):.2f}")
PY
)
                    else
                        if awk "BEGIN{exit !($BATTERY_LEVEL < 35)}"; then
                            if [ $((RANDOM % 100)) -lt 8 ]; then
                                BATTERY_LEVEL=$(python3 - <<PY
lvl=$BATTERY_LEVEL
print(f"{min(100.0, lvl + 0.4):.2f}")
PY
)
                            fi
                        fi
                    fi
                else
                    if [ $SPEED -gt 0 ]; then
                        CONS=$(python3 - <<PY
import random
speed=$SPEED
print(f"{max(0.02, min(0.10, speed/800.0)):.4f}")
PY
)
                        FUEL_LEVEL=$(python3 - <<PY
lvl=$FUEL_LEVEL
cons=$CONS
print(f"{max(0.0, lvl - cons):.2f}")
PY
)
                    else
                        if awk "BEGIN{exit !($FUEL_LEVEL < 30)}"; then
                            if [ $((RANDOM % 100)) -lt 8 ]; then
                                FUEL_LEVEL=$(python3 - <<PY
lvl=$FUEL_LEVEL
print(f"{min(100.0, lvl + 0.5):.2f}")
PY
)
                            fi
                        fi
                    fi
                fi
                if [ "$VEHICLE_TYPE" = "EV" ]; then EM=0; else EM=$((SPEED/2)); fi
                ISO_TS=$(date -u -r $ts +%Y-%m-%dT%H:%M:%SZ)
                TELEMETRY_DATA="{\"vehicle_id\": \"$vid\", \"timestamp\": \"$ISO_TS\", \"location\": {\"lat\": $LAT, \"lon\": $LON}, \"speed\": $SPEED, \"fuel_level\": $FUEL_LEVEL, \"battery_level\": $BATTERY_LEVEL, \"emissions\": $EM, \"type\": \"$VEHICLE_TYPE\", \"status\": \"active\" }"
                CODE=$(curl -s -m 5 -o /dev/null -w '%{http_code}' -X POST http://localhost:8081/api/telemetry \
                    -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d "$TELEMETRY_DATA")
                ATTEMPTED=$((ATTEMPTED+1))
                if [ "$CODE" -ge 200 ] && [ "$CODE" -lt 300 ]; then
                    COUNT=$((COUNT+1))
                fi
                if [ $((ATTEMPTED % 25)) -eq 0 ]; then progress_print "   ${WINDOW} telemetry:" "$ATTEMPTED" "$TOTAL_POINTS"; fi
                if [ "$ATTEMPTED" -ge "$TOTAL_POINTS" ]; then break; fi
            done
            if [ "$ATTEMPTED" -ge "$TOTAL_POINTS" ]; then break; fi
        done
        progress_done "   ${WINDOW} telemetry:" "$ATTEMPTED" "$TOTAL_POINTS"
        print_status "   ${WINDOW} posted: $COUNT accepted / $ATTEMPTED attempted"
    done

    # 5) Restart simulator to use all vehicles with global OSRM for proper road snapping
    print_status "Restarting simulator to activate all vehicles with global road coverage..."
    stop_simulator >/dev/null 2>&1 || true
    sleep 2
    
    # Use global OSRM for worldwide road coverage and proper snapping
    export OSRM_BASE_URL="https://router.project-osrm.org"
    export SIM_USE_EXISTING=1
    export FLEET_SIZE=4  # Use 4 vehicles maximum
    
    print_status "Starting simulator with 4 vehicles using global OSRM..."
    start_simulator global

    # 6) Verify fresh telemetry and report count (with progress)
    print_status "Verifying fresh telemetry (up to ~24s)..."
    TARGET=$VEH_CREATED
    SEEN=0
    ATTEMPTS=0
    MAX_ATTEMPTS=12
    progress_print "   Verifying telemetry:" "$ATTEMPTS" "$MAX_ATTEMPTS"
    while [ $ATTEMPTS -lt $MAX_ATTEMPTS ]; do
        FROM=$(date -u -v-30S +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d '30 seconds ago' +%Y-%m-%dT%H:%M:%SZ)
        RESP=$(curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8081/api/telemetry?from=$FROM")
        if command -v jq >/dev/null 2>&1; then
            SEEN=$(echo "$RESP" | jq -r '.[].vehicle_id' | awk 'NF' | sort | uniq | wc -l | tr -d ' ')
        else
            SEEN=$(echo "$RESP" | grep -o '"vehicle_id":"[^\"]\{24\}"' | cut -d '"' -f4 | sort | uniq | wc -l | tr -d ' ')
        fi
        progress_print "   Verifying telemetry:" "$ATTEMPTS" "$MAX_ATTEMPTS"
        echo -ne "   seen: $SEEN/$TARGET (attempt $((ATTEMPTS+1))/$MAX_ATTEMPTS)\r"
        if [ "$SEEN" -ge "$TARGET" ]; then
            break
        fi
        sleep 2
        ATTEMPTS=$((ATTEMPTS+1))
    done
    progress_done "   Verifying telemetry:" "$ATTEMPTS" "$MAX_ATTEMPTS"
    echo ""
    echo "All moving: $SEEN vehicles"

    # 7) Seed trips, maintenance, and costs for the vehicles
    print_status "Seeding trips, maintenance, and costs..."
    
    # Get fresh vehicle list after verification
    VEHICLE_IDS=()
    VEHICLE_TYPES=()
    VEHICLES_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/vehicles)
    if command -v jq >/dev/null 2>&1; then
        while read -r id type; do
            [ -n "$id" ] && VEHICLE_IDS+=("$id") && VEHICLE_TYPES+=("$type")
        done < <(echo "$VEHICLES_RESPONSE" | jq -r '.[]? | "\(.id // empty) \(.type // empty)"')
        if [ ${#VEHICLE_IDS[@]} -eq 0 ]; then 
            while read -r id type; do [ -n "$id" ] && VEHICLE_IDS+=("$id") && VEHICLE_TYPES+=("$type"); done < <(echo "$VEHICLES_RESPONSE" | jq -r '.data[]? | "\(.id // empty) \(.type // empty)"')
        fi
    fi
    if [ ${#VEHICLE_IDS[@]} -eq 0 ]; then 
        # Fallback: extract both ID and type using grep/sed
        while IFS= read -r line; do
            id=$(echo "$line" | grep -o '"id":"[a-f0-9]\{24\}"' | cut -d '"' -f4)
            type=$(echo "$line" | grep -o '"type":"[^"]*"' | cut -d '"' -f4)
            [ -n "$id" ] && VEHICLE_IDS+=("$id") && VEHICLE_TYPES+=("$type")
        done < <(echo "$VEHICLES_RESPONSE" | grep -E '"id":"[a-f0-9]{24}"' | head -10)
    fi
    
    # Seed trips (2-3 per vehicle)
    print_status "   Creating trips..."
    TRIP_CREATED=0
    TRIP_TOTAL=$(( ${#VEHICLE_IDS[@]} * 3 ))
    for vid in "${VEHICLE_IDS[@]}"; do
        for i in {1..3}; do
            TRIP_CREATED=$((TRIP_CREATED+1))
            progress_print "   Trips:" "$TRIP_CREATED" "$TRIP_TOTAL"
            
            # Generate trip data
            START_TIME=$(date -u -v-$((i*8))H +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d "$((i*8)) hours ago" +%Y-%m-%dT%H:%M:%SZ)
            END_TIME=$(date -u -v-$((i*8-2))H +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d "$((i*8-2)) hours ago" +%Y-%m-%dT%H:%M:%SZ)
            
            # Random start/end locations
            START_LAT=$(python3 - <<PY
import random
print(f"{random.uniform(40.7, 40.8):.6f}")
PY
)
            START_LON=$(python3 - <<PY
import random
print(f"{random.uniform(-74.1, -74.0):.6f}")
PY
)
            END_LAT=$(python3 - <<PY
import random
print(f"{random.uniform(40.7, 40.8):.6f}")
PY
)
            END_LON=$(python3 - <<PY
import random
print(f"{random.uniform(-74.1, -74.0):.6f}")
PY
)
            
            DISTANCE=$(python3 - <<PY
import random
print(f"{random.uniform(5.0, 25.0):.2f}")
PY
)
            DURATION=$(python3 - <<PY
import random
print(f"{random.uniform(0.5, 2.0):.2f}")
PY
)
            FUEL_CONSUMPTION=$(python3 - <<PY
import random
print(f"{random.uniform(2.0, 8.0):.2f}")
PY
)
            BATTERY_CONSUMPTION=$(python3 - <<PY
import random
print(f"{random.uniform(5.0, 15.0):.2f}")
PY
)
            COST=$(python3 - <<PY
import random
print(f"{random.uniform(10.0, 50.0):.2f}")
PY
)
            
            PURPOSES=("business" "personal" "delivery")
            PURPOSE=${PURPOSES[$((RANDOM % ${#PURPOSES[@]}))]}
            
            TRIP_DATA=$(cat <<JSON
{
  "vehicle_id": "$vid",
  "driver_id": "driver_$((RANDOM % 1000))",
  "start_location": {"lat": $START_LAT, "lon": $START_LON},
  "end_location": {"lat": $END_LAT, "lon": $END_LON},
  "start_time": "$START_TIME",
  "end_time": "$END_TIME",
  "distance": $DISTANCE,
  "duration": $DURATION,
  "fuel_consumption": $FUEL_CONSUMPTION,
  "battery_consumption": $BATTERY_CONSUMPTION,
  "cost": $COST,
  "purpose": "$PURPOSE",
  "status": "completed",
  "notes": "Auto-generated trip data"
}
JSON
)
            
            CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8081/api/trips \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $TOKEN" \
                -d "$TRIP_DATA")
            if [ "$CODE" -ge 200 ] && [ "$CODE" -lt 300 ]; then
                # Success
                :
            fi
        done
    done
    progress_done "   Trips:" "$TRIP_CREATED" "$TRIP_TOTAL"
    
    # Seed maintenance records (2-3 per vehicle)
    print_status "   Creating maintenance records..."
    MAINT_CREATED=0
    MAINT_TOTAL=$(( ${#VEHICLE_IDS[@]} * 3 ))
    for vid in "${VEHICLE_IDS[@]}"; do
        for i in {1..3}; do
            MAINT_CREATED=$((MAINT_CREATED+1))
            progress_print "   Maintenance:" "$MAINT_CREATED" "$MAINT_TOTAL"
            
            # Generate maintenance data
            SERVICE_TYPES=("oil_change" "tire_rotation" "brake_service" "battery_service" "inspection")
            SERVICE_TYPE=${SERVICE_TYPES[$((RANDOM % ${#SERVICE_TYPES[@]}))]}
            
            SERVICE_DATE=$(date -u -v-$((i*30))d +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d "$((i*30)) days ago" +%Y-%m-%dT%H:%M:%SZ)
            NEXT_SERVICE_DATE=$(date -u -v+$((90+i*30))d +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d "+$((90+i*30)) days" +%Y-%m-%dT%H:%M:%SZ)
            
            MILEAGE=$(python3 - <<PY
import random
print(f"{random.uniform(10000.0, 50000.0):.1f}")
PY
)
            COST=$(python3 - <<PY
import random
print(f"{random.uniform(50.0, 300.0):.2f}")
PY
)
            LABOR_COST=$(python3 - <<PY
import random
print(f"{random.uniform(20.0, 150.0):.2f}")
PY
)
            PARTS_COST=$(python3 - <<PY
import random
print(f"{random.uniform(30.0, 200.0):.2f}")
PY
)
            
            TECHNICIANS=("John Smith" "Sarah Johnson" "Mike Wilson" "Lisa Brown" "David Davis")
            TECHNICIAN=${TECHNICIANS[$((RANDOM % ${#TECHNICIANS[@]}))]}
            
            LOCATIONS=("Downtown Service Center" "Main Street Garage" "Fleet Maintenance Hub" "Auto Care Plus")
            LOCATION=${LOCATIONS[$((RANDOM % ${#LOCATIONS[@]}))]}
            
            PRIORITIES=("low" "medium" "high")
            PRIORITY=${PRIORITIES[$((RANDOM % ${#PRIORITIES[@]}))]}
            
            MAINT_DATA=$(cat <<JSON
{
  "vehicle_id": "$vid",
  "service_type": "$SERVICE_TYPE",
  "description": "Routine $SERVICE_TYPE service",
  "service_date": "$SERVICE_DATE",
  "next_service_date": "$NEXT_SERVICE_DATE",
  "mileage": $MILEAGE,
  "cost": $COST,
  "labor_cost": $LABOR_COST,
  "parts_cost": $PARTS_COST,
  "technician": "$TECHNICIAN",
  "service_location": "$LOCATION",
  "status": "completed",
  "priority": "$PRIORITY",
  "notes": "Auto-generated maintenance record"
}
JSON
)
            
            CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8081/api/maintenance \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $TOKEN" \
                -d "$MAINT_DATA")
            if [ "$CODE" -ge 200 ] && [ "$CODE" -lt 300 ]; then
                # Success
                :
            fi
        done
    done
    progress_done "   Maintenance:" "$MAINT_CREATED" "$MAINT_TOTAL"
    
    # Seed cost records (3-4 per vehicle)
    print_status "   Creating cost records..."
    COST_CREATED=0
    COST_TOTAL=$(( ${#VEHICLE_IDS[@]} * 4 ))
    for vid in "${VEHICLE_IDS[@]}"; do
        for i in {1..4}; do
            COST_CREATED=$((COST_CREATED+1))
            progress_print "   Costs:" "$COST_CREATED" "$COST_TOTAL"
            
            # Generate cost data
            CATEGORIES=("fuel" "maintenance" "insurance" "registration" "tolls" "parking" "other")
            CATEGORY=${CATEGORIES[$((RANDOM % ${#CATEGORIES[@]}))]}
            
            COST_DATE=$(date -u -v-$((i*15))d +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -d "$((i*15)) days ago" +%Y-%m-%dT%H:%M:%SZ)
            
            AMOUNT=$(python3 - <<PY
import random
if "$CATEGORY" == "fuel":
    print(f"{random.uniform(30.0, 80.0):.2f}")
elif "$CATEGORY" == "maintenance":
    print(f"{random.uniform(50.0, 300.0):.2f}")
elif "$CATEGORY" == "insurance":
    print(f"{random.uniform(100.0, 200.0):.2f}")
elif "$CATEGORY" == "registration":
    print(f"{random.uniform(50.0, 150.0):.2f}")
elif "$CATEGORY" == "tolls":
    print(f"{random.uniform(5.0, 25.0):.2f}")
elif "$CATEGORY" == "parking":
    print(f"{random.uniform(10.0, 40.0):.2f}")
else:
    print(f"{random.uniform(20.0, 100.0):.2f}")
PY
)
            
            INVOICE_NUM=$(python3 - <<PY
import random
print(f"INV-{random.randint(10000, 99999)}")
PY
)
            
            VENDORS=("Shell" "ExxonMobil" "BP" "Chevron" "AutoZone" "NAPA" "O'Reilly" "Fleet Services Inc")
            VENDOR=${VENDORS[$((RANDOM % ${#VENDORS[@]}))]}
            
            LOCATIONS=("Downtown Station" "Highway 101" "Main Street" "Fleet Depot")
            LOCATION=${LOCATIONS[$((RANDOM % ${#LOCATIONS[@]}))]}
            
            PAYMENT_METHODS=("credit_card" "cash" "check" "electronic")
            PAYMENT_METHOD=${PAYMENT_METHODS[$((RANDOM % ${#PAYMENT_METHODS[@]}))]}
            
            STATUSES=("pending" "paid" "disputed")
            STATUS=${STATUSES[$((RANDOM % ${#STATUSES[@]}))]}
            
            COST_DATA=$(cat <<JSON
{
  "vehicle_id": "$vid",
  "category": "$CATEGORY",
  "description": "$CATEGORY expense for vehicle $vid",
  "amount": $AMOUNT,
  "date": "$COST_DATE",
  "invoice_number": "$INVOICE_NUM",
  "vendor": "$VENDOR",
  "location": "$LOCATION",
  "payment_method": "$PAYMENT_METHOD",
  "status": "$STATUS",
  "notes": "Auto-generated cost record"
}
JSON
)
            
            CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8081/api/costs \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $TOKEN" \
                -d "$COST_DATA")
            if [ "$CODE" -ge 200 ] && [ "$CODE" -lt 300 ]; then
                # Success
                :
            fi
        done
    done
    progress_done "   Costs:" "$COST_CREATED" "$COST_TOTAL"
    
    print_status "   Data seeding complete: trips, maintenance, and costs created"

    # 8) Prune any vehicles without telemetry to keep fleet tab clean
    print_status "Pruning vehicles without telemetry..."
    VEH_LIST=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/vehicles)
    TO_DELETE=0
    if command -v jq >/dev/null 2>&1; then
        while read -r vid; do
            [ -z "$vid" ] && continue
            # Check if there is at least one telemetry record
            TJSON=$(curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:8081/api/telemetry?vehicle_id=$vid&limit=1&sort=desc")
            HAS=$(echo "$TJSON" | jq -r 'length')
            if [ "$HAS" = "0" ] || [ -z "$HAS" ]; then
                CODE=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE -H "Authorization: Bearer $TOKEN" "http://localhost:8081/api/vehicles/$vid")
                if [ "$CODE" -ge 200 ] && [ "$CODE" -lt 300 ]; then TO_DELETE=$((TO_DELETE+1)); fi
            fi
        done < <(echo "$VEH_LIST" | jq -r '.[]? | .id // empty')
        if [ "$TO_DELETE" -gt 0 ]; then
            print_status "Removed $TO_DELETE vehicles without telemetry"
        else
            print_status "No vehicles without telemetry found"
        fi
    else
        # Fallback (less robust without jq)
        :
    fi
}

# Helper: snap a lat lon to nearest road via OSRM (prints "lat lon")
osrm_snap() {
    local lat="$1"; local lon="$2"
    if [ -z "$OSRM_EFF_URL" ]; then echo "$lat $lon"; return 0; fi
    local url="$OSRM_EFF_URL/nearest/v1/driving/${lon},${lat}?number=1"
    local out=""
    if command -v jq >/dev/null 2>&1; then
        out=$(curl -s "$url" | jq -r 'if (.waypoints and .waypoints[0] and (.waypoints[0].location|type)=="array" and (.waypoints[0].location|length)>=2) then "\(.waypoints[0].location[1]) \(.waypoints[0].location[0])" else empty end')
    else
        out=$(curl -s "$url" | sed -n 's/.*\[\([-0-9.]*\),\([-0-9.]*\)\].*/\2 \1/p' | head -n1)
    fi
    if echo "$out" | grep -Eq '^-?[0-9]+\.?[0-9]*[[:space:]]+-?[0-9]+\.?[0-9]*$'; then
        echo "$out"
    else
        echo "$lat $lon"
    fi
}

# Main script logic
case "${1:-}" in
    "start")
        start_fleet_sustainability
        ;;
    "stop")
        stop_fleet_sustainability
        ;;
    "status")
        show_status
        ;;
    "restart")
        stop_fleet_sustainability
        sleep 2
        start_fleet_sustainability
        ;;
    "populate")
        populate_database
        ;;
    "clear")
        clear_database
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    "troubleshoot")
        troubleshooting
        ;;
    "sim-start")
        # Allow SIM_GLOBAL or arg 'global'
        MODE="${2:-${SIM_GLOBAL:+global}}"
        if [ "$MODE" = "global" ]; then
            start_simulator global
        else
            start_simulator local
        fi
        ;;
    "sim-stop")
        stop_simulator
        ;;
    "sim-status")
        simulator_status
        ;;
    "auto-fix")
        auto_fix
        ;;
    "osrm-start")
        start_local_osrm
        ;;
    "osrm-stop")
        stop_local_osrm
        ;;
    "osrm-status")
        osrm_status
        ;;
    "")
        while true; do
            echo ""
            print_header
            echo ""
            echo "Welcome to Fleet Sustainability Manager!"
            echo ""
            echo "What would you like to do?"
            echo ""
            echo "1) Start the application"
            echo "2) Stop the application"
            echo "3) Check status"
            echo "4) Restart the application"
            echo "5) Populate database with dummy data"
            echo "6) Clear database data (preserves users)"
            echo "7) Simulator"
            echo "8) OSRM"
            echo "9) Troubleshoot"
            echo "10) Show help"
            echo ""
            read -p "Enter your choice (1-10): " choice

            case $choice in
                1)
                    start_fleet_sustainability
                    ;;
                2)
                    stop_fleet_sustainability
                    ;;
                3)
                    show_status
                    ;;
                4)
                    stop_fleet_sustainability
                    sleep 2
                    start_fleet_sustainability
                    ;;
                5)
                    populate_database
                    ;;
                6)
                    clear_database
                    ;;
                7)
                    simulator_menu
                    ;;
                8)
                    osrm_menu
                    ;;
                9)
                    troubleshooting
                    ;;
                10)
                    show_help
                    ;;
                *)
                    print_error "Invalid choice. Please run '$0 help' for options."
                    sleep 2
                    ;;
            esac
        done
        ;;
    *)
        print_error "Unknown option: $1"
        echo ""
        show_help
        exit 1
        ;;
esac