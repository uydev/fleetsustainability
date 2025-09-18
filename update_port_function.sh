#!/bin/bash

# Script to update the free_port function in fleet_sustainability.sh

# Create a backup
cp scripts/fleet_sustainability.sh scripts/fleet_sustainability.sh.backup

# Create the new free_port function
cat > /tmp/new_free_port.sh << 'EOF'
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
EOF

# Replace the function in the main script
# Find the start and end of the free_port function
start_line=$(grep -n "^free_port()" scripts/fleet_sustainability.sh | cut -d: -f1)
end_line=$(grep -n "^# Function to check Docker containers" scripts/fleet_sustainability.sh | cut -d: -f1)

if [ -n "$start_line" ] && [ -n "$end_line" ]; then
    # Create new file with updated function
    head -n $((start_line - 1)) scripts/fleet_sustainability.sh > /tmp/script_part1.sh
    cat /tmp/new_free_port.sh > /tmp/script_part2.sh
    tail -n +$end_line scripts/fleet_sustainability.sh > /tmp/script_part3.sh
    
    # Combine the parts
    cat /tmp/script_part1.sh /tmp/script_part2.sh /tmp/script_part3.sh > scripts/fleet_sustainability.sh
    
    echo "✅ Successfully updated free_port function in fleet_sustainability.sh"
    echo "📁 Backup created as fleet_sustainability.sh.backup"
    
    # Clean up temp files
    rm -f /tmp/new_free_port.sh /tmp/script_part1.sh /tmp/script_part2.sh /tmp/script_part3.sh
else
    echo "❌ Could not find free_port function boundaries"
    exit 1
fi
