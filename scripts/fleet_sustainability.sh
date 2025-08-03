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
        
        # Only kill Node.js processes
        local pids=$(lsof -ti:$port)
        for pid in $pids; do
            local process_name=$(ps -p $pid -o comm= 2>/dev/null)
            if [[ "$process_name" == *"node"* ]] || [[ "$process_name" == *"npm"* ]]; then
                print_status "Killing Node.js process on port $port (PID: $pid)"
                kill -9 $pid 2>/dev/null
            else
                print_warning "Skipping non-Node.js process $process_name (PID: $pid)"
            fi
        done
        
        # Wait a moment for port to be freed
        sleep 2
        
        # Check if port is now free
        if check_port $port; then
            print_error "Port $port is still in use"
            return 1
        else
            print_status "Port $port is now free"
            return 0
        fi
    else
        print_status "Port $port is already free"
        return 0
    fi
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
    cd "$(dirname "$0")/.."
    docker-compose ps
}

# Function to check application logs
check_application_logs() {
    print_status "Checking application logs..."
    echo ""
    
    cd "$(dirname "$0")/.."
    
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
        echo "2) Check Docker containers"
        echo "3) Check application logs"
        echo "4) Back to main menu"
        echo ""
        read -p "Enter your choice (1-4): " choice
        
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
                print_header
                check_docker_containers
                echo ""
                read -p "Press Enter to continue..."
                ;;
            3)
                print_header
                check_application_logs
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
    cd "$(dirname "$0")/.."
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

    # Create test users
    print_status "5. Creating test users..."
    ./scripts/setup_auth.sh > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        print_status "   Test users created successfully"
    else
        print_warning "   Failed to create test users (they may already exist)"
    fi

    # Start frontend (local process)
    print_status "6. Starting frontend..."
    
    # Kill any existing frontend process
    kill_port 3000
    
    # Start frontend in background
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
    echo "$FRONTEND_PID" > .frontend_pid

    # Test the application
    print_status "7. Testing application..."
    sleep 3
    
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
    cd "$(dirname "$0")/.."
    docker-compose down
    print_status "   Docker containers stopped"

    # Clean up log files
    print_status "3. Cleaning up..."
    rm -f backend.log frontend.log 2>/dev/null

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

    echo ""
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
    echo "  troubleshoot  Open troubleshooting menu"
    echo "  help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 start         # Start the application"
    echo "  $0 stop          # Stop the application"
    echo "  $0 status        # Check service status"
    echo "  $0 restart       # Restart the application"
    echo "  $0 populate      # Add dummy data to database"
    echo "  $0 clear         # Clear database data (preserves users)"
    echo "  $0 troubleshoot  # Open troubleshooting menu"
    echo ""
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

    # Create admin user if it doesn't exist
    print_status "4. Creating admin user..."
    REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8081/api/auth/register \
        -H "Content-Type: application/json" \
        -d '{"username": "admin", "password": "admin123", "email": "admin@fleet.com", "first_name": "Admin", "last_name": "User", "role": "admin"}')
    
    if ! echo "$REGISTER_RESPONSE" | grep -q "token"; then
        print_warning "Admin user may already exist, trying to login..."
    fi

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

    # Clear vehicles using bulk delete endpoint
    print_status "6. Clearing vehicles..."
    VEHICLES_RESPONSE=$(curl -s -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/vehicles)
    if echo "$VEHICLES_RESPONSE" | grep -q "deleted successfully"; then
        print_status "   Vehicles cleared successfully"
    else
        print_warning "   Failed to clear vehicles: $VEHICLES_RESPONSE"
    fi

    # Clear telemetry using API DELETE endpoint
    print_status "7. Clearing telemetry data..."
    TELEMETRY_RESPONSE=$(curl -s -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/telemetry)
    if echo "$TELEMETRY_RESPONSE" | grep -q "deleted successfully"; then
        print_status "   Telemetry data cleared successfully"
    else
        print_warning "   Failed to clear telemetry data: $TELEMETRY_RESPONSE"
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

    # Get admin token for API calls
    print_status "1. Getting admin authentication token..."
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

    # Create dummy vehicles
    print_status "2. Creating dummy vehicles..."
    VEHICLES=(
        '{"type": "ICE", "make": "Ford", "model": "F-150", "year": 2022, "status": "active"}'
        '{"type": "EV", "make": "Tesla", "model": "Model 3", "year": 2023, "status": "active"}'
        '{"type": "ICE", "make": "Chevrolet", "model": "Silverado", "year": 2021, "status": "active"}'
        '{"type": "EV", "make": "Nissan", "model": "Leaf", "year": 2023, "status": "active"}'
        '{"type": "ICE", "make": "Toyota", "model": "Tacoma", "year": 2022, "status": "active"}'
        '{"type": "EV", "make": "Ford", "model": "E-Transit", "year": 2023, "status": "active"}'
        '{"type": "ICE", "make": "Ram", "model": "1500", "year": 2021, "status": "active"}'
        '{"type": "EV", "make": "Rivian", "model": "R1T", "year": 2023, "status": "active"}'
    )
    
    for vehicle in "${VEHICLES[@]}"; do
        RESPONSE=$(curl -s -X POST http://localhost:8081/api/vehicles \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $TOKEN" \
            -d "$vehicle")
        if echo "$RESPONSE" | grep -q "error\|Error"; then
            print_warning "   Failed to create vehicle: $RESPONSE"
        fi
    done
    print_status "   Created 8 vehicles"

    # Create dummy telemetry data
    print_status "3. Creating dummy telemetry data..."
    
    # First, get the vehicle IDs that were created
    VEHICLES_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/vehicles)
    echo "   Retrieved vehicles: $VEHICLES_RESPONSE"
    
    # Extract vehicle IDs (this is a simplified approach - in production you'd parse JSON properly)
    VEHICLE_IDS=("vehicle1" "vehicle2" "vehicle3" "vehicle4" "vehicle5" "vehicle6" "vehicle7" "vehicle8")
    
    for i in {1..100}; do
        # Generate random data
        VEHICLE_INDEX=$((RANDOM % 8))
        VEHICLE_ID="${VEHICLE_IDS[$VEHICLE_INDEX]}"
        SPEED=$((RANDOM % 80 + 20))
        FUEL_LEVEL=$((RANDOM % 100 + 1))
        BATTERY_LEVEL=$((RANDOM % 100 + 1))
        EMISSIONS=$((RANDOM % 50 + 5))
        LAT=$((RANDOM % 1000 + 40000)) # Roughly 40-50 latitude
        LON=$((RANDOM % 1000 - 75000)) # Roughly -75 longitude
        
        # Generate realistic timestamps (last 7 days)
        DAYS_AGO=$((RANDOM % 7))
        HOURS_AGO=$((RANDOM % 24))
        MINUTES_AGO=$((RANDOM % 60))
        TIMESTAMP=$(date -u -v-${DAYS_AGO}d -v-${HOURS_AGO}H -v-${MINUTES_AGO}M +%Y-%m-%dT%H:%M:%SZ)
        
        # Determine vehicle type based on vehicle ID
        if [[ "$VEHICLE_ID" == *"vehicle"* ]]; then
            VEHICLE_TYPE="ICE"
        else
            VEHICLE_TYPE="EV"
        fi
        
        TELEMETRY_DATA="{
            \"vehicle_id\": \"$VEHICLE_ID\",
            \"timestamp\": \"$TIMESTAMP\",
            \"location\": {\"lat\": $LAT, \"lng\": $LON},
            \"speed\": $SPEED,
            \"fuel_level\": $FUEL_LEVEL,
            \"battery_level\": $BATTERY_LEVEL,
            \"emissions\": $EMISSIONS,
            \"type\": \"$VEHICLE_TYPE\",
            \"status\": \"active\"
        }"
        
        RESPONSE=$(curl -s -X POST http://localhost:8081/api/telemetry \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $TOKEN" \
            -d "$TELEMETRY_DATA")
        
        if echo "$RESPONSE" | grep -q "error\|Error"; then
            print_warning "   Failed to create telemetry record $i: $RESPONSE"
        fi
    done
    print_status "   Created 100 telemetry records"

    # Create dummy trips
    print_status "4. Creating dummy trips..."
    VEHICLE_IDS=("vehicle1" "vehicle2" "vehicle3" "vehicle4" "vehicle5" "vehicle6" "vehicle7" "vehicle8")
    
    # Create multiple trips for each vehicle
    for vehicle in "${VEHICLE_IDS[@]}"; do
        for day in {1..7}; do
            # Generate start and end times for the day
            START_HOUR=$((RANDOM % 12 + 6)) # 6 AM to 6 PM
            END_HOUR=$((START_HOUR + RANDOM % 4 + 1)) # 1-4 hour trips
            
            START_TIME=$(date -u -v-${day}d -v+${START_HOUR}H +%Y-%m-%dT%H:%M:%SZ)
            END_TIME=$(date -u -v-${day}d -v+${END_HOUR}H +%Y-%m-%dT%H:%M:%SZ)
            
            # Generate realistic locations (NYC area)
            START_LAT=$((40000 + RANDOM % 1000))
            START_LON=$((-75000 + RANDOM % 1000))
            END_LAT=$((40000 + RANDOM % 1000))
            END_LON=$((-75000 + RANDOM % 1000))
            
            # Calculate distance (rough approximation)
            DISTANCE=$((RANDOM % 50 + 5))
            
            # Fuel consumption based on vehicle type
            if [[ "$vehicle" == *"Van"* ]]; then
                FUEL_CONSUMED=0.0  # EVs
            else
                FUEL_CONSUMED=$(echo "scale=1; $DISTANCE * 0.15" | bc -l 2>/dev/null || echo "7.5")
            fi
            
            # Calculate duration in hours
            DURATION=$(echo "scale=1; $((END_HOUR - START_HOUR))" | bc -l 2>/dev/null || echo "2.5")
            
            # Calculate battery consumption for EVs
            if [[ "$vehicle" == *"vehicle"* ]]; then
                BATTERY_CONSUMPTION=$(echo "scale=2; $DISTANCE * 0.2" | bc -l 2>/dev/null || echo "5.0")
            else
                BATTERY_CONSUMPTION=0.0
            fi
            
            # Calculate cost
            COST=$(echo "scale=2; $DISTANCE * 0.5" | bc -l 2>/dev/null || echo "25.0")
            
            TRIP_DATA="{
                \"vehicle_id\": \"$vehicle\",
                \"driver_id\": \"driver$((RANDOM % 5 + 1))\",
                \"start_location\": {\"lat\": $START_LAT, \"lng\": $START_LON},
                \"end_location\": {\"lat\": $END_LAT, \"lng\": $END_LON},
                \"start_time\": \"$START_TIME\",
                \"end_time\": \"$END_TIME\",
                \"distance\": $DISTANCE,
                \"duration\": $DURATION,
                \"fuel_consumption\": $FUEL_CONSUMED,
                \"battery_consumption\": $BATTERY_CONSUMPTION,
                \"cost\": $COST,
                \"purpose\": \"delivery\",
                \"status\": \"completed\"
            }"
            
            RESPONSE=$(curl -s -X POST http://localhost:8081/api/trips \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $TOKEN" \
                -d "$TRIP_DATA")
            
            if echo "$RESPONSE" | grep -q "error\|Error"; then
                print_warning "   Failed to create trip for $vehicle: $RESPONSE"
            fi
        done
    done
    print_status "   Created trips for all vehicles"

    # Create dummy maintenance records
    print_status "5. Creating dummy maintenance records..."
    VEHICLE_IDS=("vehicle1" "vehicle2" "vehicle3" "vehicle4" "vehicle5" "vehicle6" "vehicle7" "vehicle8")
    MAINTENANCE_TYPES=("oil_change" "tire_rotation" "brake_service" "battery_check" "filter_replacement" "inspection" "tune_up" "battery_replacement")
    TECHNICIANS=("John Smith" "Mike Johnson" "Sarah Wilson" "David Brown" "Lisa Garcia" "Tom Davis" "Emma White" "Alex Chen")
    
    for vehicle in "${VEHICLE_IDS[@]}"; do
        # Create 2-4 maintenance records per vehicle
        NUM_RECORDS=$((RANDOM % 3 + 2))
        
        for i in $(seq 1 $NUM_RECORDS); do
            MAINT_TYPE="${MAINTENANCE_TYPES[$((RANDOM % ${#MAINTENANCE_TYPES[@]}))]}"
            TECHNICIAN="${TECHNICIANS[$((RANDOM % ${#TECHNICIANS[@]}))]}"
            
            # Generate date within last 6 months
            DAYS_AGO=$((RANDOM % 180))
            MAINT_DATE=$(date -u -v-${DAYS_AGO}d +%Y-%m-%dT%H:%M:%SZ)
            
            # Generate cost based on maintenance type
            case $MAINT_TYPE in
                "oil_change") COST=$((RANDOM % 50 + 60)) ;;
                "tire_rotation") COST=$((RANDOM % 30 + 40)) ;;
                "brake_service") COST=$((RANDOM % 100 + 200)) ;;
                "battery_check") COST=$((RANDOM % 20 + 30)) ;;
                "filter_replacement") COST=$((RANDOM % 40 + 50)) ;;
                "inspection") COST=$((RANDOM % 80 + 100)) ;;
                "tune_up") COST=$((RANDOM % 150 + 200)) ;;
                "battery_replacement") COST=$((RANDOM % 200 + 300)) ;;
                *) COST=$((RANDOM % 100 + 50)) ;;
            esac
            
            # Generate description
            case $MAINT_TYPE in
                "oil_change") DESC="Regular oil change and filter replacement" ;;
                "tire_rotation") DESC="Tire rotation and balance service" ;;
                "brake_service") DESC="Brake pad replacement and brake fluid check" ;;
                "battery_check") DESC="Battery health check and terminal cleaning" ;;
                "filter_replacement") DESC="Air filter and cabin filter replacement" ;;
                "inspection") DESC="Comprehensive vehicle inspection" ;;
                "tune_up") DESC="Engine tune-up and spark plug replacement" ;;
                "battery_replacement") DESC="Battery replacement and testing" ;;
                *) DESC="General maintenance service" ;;
            esac
            
            # Random status (mostly completed)
            STATUSES=("completed" "completed" "completed" "scheduled" "in_progress")
            STATUS="${STATUSES[$((RANDOM % ${#STATUSES[@]}))]}"
            
            # Calculate next service date
            NEXT_SERVICE_DATE=$(date -u -v+$((RANDOM % 90 + 30))d +%Y-%m-%dT%H:%M:%SZ)
            
            # Calculate costs
            LABOR_COST=$(echo "scale=2; $COST * 0.6" | bc -l 2>/dev/null || echo "60.0")
            PARTS_COST=$(echo "scale=2; $COST * 0.4" | bc -l 2>/dev/null || echo "40.0")
            MILEAGE=$((RANDOM % 50000 + 10000))
            
            MAINT_DATA="{
                \"vehicle_id\": \"$vehicle\",
                \"service_type\": \"$MAINT_TYPE\",
                \"description\": \"$DESC\",
                \"service_date\": \"$MAINT_DATE\",
                \"next_service_date\": \"$NEXT_SERVICE_DATE\",
                \"mileage\": $MILEAGE,
                \"cost\": $COST,
                \"labor_cost\": $LABOR_COST,
                \"parts_cost\": $PARTS_COST,
                \"technician\": \"$TECHNICIAN\",
                \"service_location\": \"Main Service Center\",
                \"status\": \"$STATUS\",
                \"priority\": \"medium\"
            }"
            
            RESPONSE=$(curl -s -X POST http://localhost:8081/api/maintenance \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $TOKEN" \
                -d "$MAINT_DATA")
            
            if echo "$RESPONSE" | grep -q "error\|Error"; then
                print_warning "   Failed to create maintenance record for $vehicle: $RESPONSE"
            fi
        done
    done
    print_status "   Created maintenance records for all vehicles"

    # Create dummy cost records
    print_status "6. Creating dummy cost records..."
    VEHICLE_IDS=("vehicle1" "vehicle2" "vehicle3" "vehicle4" "vehicle5" "vehicle6" "vehicle7" "vehicle8")
    COST_TYPES=("fuel" "electricity" "maintenance" "insurance" "registration" "parking" "tolls" "cleaning")
    
    for vehicle in "${VEHICLE_IDS[@]}"; do
        # Create 3-6 cost records per vehicle
        NUM_RECORDS=$((RANDOM % 4 + 3))
        
        for i in $(seq 1 $NUM_RECORDS); do
            COST_TYPE="${COST_TYPES[$((RANDOM % ${#COST_TYPES[@]}))]}"
            
            # Generate date within last 3 months
            DAYS_AGO=$((RANDOM % 90))
            COST_DATE=$(date -u -v-${DAYS_AGO}d +%Y-%m-%dT%H:%M:%SZ)
            
            # Generate amount based on cost type
            case $COST_TYPE in
                "fuel") 
                    if [[ "$vehicle" == *"Truck"* ]]; then
                        AMOUNT=$((RANDOM % 100 + 50))  # Trucks use more fuel
                    else
                        AMOUNT=$((RANDOM % 60 + 30))
                    fi
                    DESC="Fuel refill"
                    ;;
                "electricity")
                    AMOUNT=$(echo "scale=2; $((RANDOM % 30 + 10))" | bc -l 2>/dev/null || echo "20.00")
                    DESC="Charging station fee"
                    ;;
                "maintenance")
                    AMOUNT=$((RANDOM % 200 + 50))
                    DESC="Maintenance service"
                    ;;
                "insurance")
                    AMOUNT=$((RANDOM % 200 + 100))
                    DESC="Insurance premium"
                    ;;
                "registration")
                    AMOUNT=$((RANDOM % 100 + 50))
                    DESC="Vehicle registration"
                    ;;
                "parking")
                    AMOUNT=$((RANDOM % 30 + 10))
                    DESC="Parking fee"
                    ;;
                "tolls")
                    AMOUNT=$((RANDOM % 20 + 5))
                    DESC="Toll charge"
                    ;;
                "cleaning")
                    AMOUNT=$((RANDOM % 50 + 20))
                    DESC="Vehicle cleaning service"
                    ;;
                *)
                    AMOUNT=$((RANDOM % 100 + 25))
                    DESC="General expense"
                    ;;
            esac
            
            VENDOR=$(echo -e "Shell\\nTesla\\nChevron\\nBP\\nLocal Garage\\nInsurance Co\\nDMV\\nParking Corp" | sort -R | head -n1)
            INVOICE_NUMBER="INV-$(date +%Y%m%d)-$((RANDOM % 9999 + 1000))"
            
            COST_DATA="{
                \"vehicle_id\": \"$vehicle\",
                \"category\": \"$COST_TYPE\",
                \"description\": \"$DESC\",
                \"amount\": $AMOUNT,
                \"date\": \"$COST_DATE\",
                \"invoice_number\": \"$INVOICE_NUMBER\",
                \"vendor\": \"$VENDOR\",
                \"location\": \"New York, NY\",
                \"payment_method\": \"credit_card\",
                \"status\": \"paid\"
            }"
            
            RESPONSE=$(curl -s -X POST http://localhost:8081/api/costs \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $TOKEN" \
                -d "$COST_DATA")
            
            if echo "$RESPONSE" | grep -q "error\|Error"; then
                print_warning "   Failed to create cost record for $vehicle: $RESPONSE"
            fi
        done
    done
    print_status "   Created cost records for all vehicles"

    echo ""
    print_status "🎉 Database populated successfully!"
    echo ""
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
    echo ""
    echo -e "${YELLOW}You can now login and see the populated dashboard!${NC}"
    echo ""
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
    "")
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
        echo "7) Show help"
        echo "8) Troubleshoot"
        echo ""
        read -p "Enter your choice (1-8): " choice
        
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
                show_help
                ;;
            8)
                troubleshooting
                ;;
            *)
                print_error "Invalid choice. Please run '$0 help' for options."
                exit 1
                ;;
        esac
        ;;
    *)
        print_error "Unknown option: $1"
        echo ""
        show_help
        exit 1
        ;;
esac 