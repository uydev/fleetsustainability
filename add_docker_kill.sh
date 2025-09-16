#!/bin/bash

# Script to add Docker killing function to fleet_sustainability.sh

# Find the line number after check_docker_containers function
end_line=$(grep -n "^# Function to show troubleshooting menu" scripts/fleet_sustainability.sh | cut -d: -f1)

if [ -z "$end_line" ]; then
    echo "❌ Could not find troubleshooting menu location"
    exit 1
fi

# Create the Docker killing function
cat > /tmp/docker_kill_function.sh << 'EOF'

# Function to force kill Docker engine and processes
kill_docker_engine() {
    print_header
    print_status "Force killing Docker engine and all Docker processes..."
    echo ""
    
    # Stop Docker Desktop application
    print_status "Stopping Docker Desktop application..."
    killall "Docker Desktop" 2>/dev/null || true
    killall "Docker" 2>/dev/null || true
    killall "com.docker.backend" 2>/dev/null || true
    
    # Stop Docker daemon
    print_status "Stopping Docker daemon..."
    sudo launchctl stop com.docker.docker 2>/dev/null || true
    
    # Kill all Docker processes
    print_status "Killing all Docker processes..."
    sudo pkill -f docker 2>/dev/null || true
    sudo pkill -f com.docker 2>/dev/null || true
    
    # Force kill any remaining Docker processes
    print_status "Force killing remaining Docker processes..."
    sudo pkill -9 -f docker 2>/dev/null || true
    sudo pkill -9 -f com.docker 2>/dev/null || true
    
    # Wait for processes to die
    sleep 3
    
    # Check if Docker is still running
    if docker info >/dev/null 2>&1; then
        print_warning "Docker is still running, trying more aggressive approach..."
        sudo pkill -9 -f "Docker Desktop" 2>/dev/null || true
        sudo pkill -9 -f "com.docker" 2>/dev/null || true
    else
        print_status "✅ Docker engine successfully stopped"
    fi
    
    echo ""
    print_status "🎉 Docker engine kill complete!"
    echo ""
    read -p "Press Enter to continue..."
}

EOF

# Insert the function before the troubleshooting menu
head -n $((end_line - 1)) scripts/fleet_sustainability.sh > /tmp/script_part1.sh
cat /tmp/docker_kill_function.sh >> /tmp/script_part1.sh
tail -n +$end_line scripts/fleet_sustainability.sh > /tmp/script_part2.sh

# Combine the parts
cat /tmp/script_part1.sh /tmp/script_part2.sh > scripts/fleet_sustainability.sh

echo "✅ Successfully added Docker killing function to fleet_sustainability.sh"

# Clean up temp files
rm -f /tmp/docker_kill_function.sh /tmp/script_part1.sh /tmp/script_part2.sh
