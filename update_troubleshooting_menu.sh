#!/bin/bash

# Script to update troubleshooting menu to include Docker killing option

# Find the troubleshooting menu section
start_line=$(grep -n "echo \"1) Free ports" scripts/fleet_sustainability.sh | cut -d: -f1)
end_line=$(grep -n "echo \"5) Back to main menu" scripts/fleet_sustainability.sh | cut -d: -f1)

if [ -z "$start_line" ] || [ -z "$end_line" ]; then
    echo "❌ Could not find troubleshooting menu boundaries"
    exit 1
fi

# Create updated menu
cat > /tmp/updated_menu.sh << 'EOF'
        echo "1) Free ports (8080, 8081, 8082, 3000)"
        echo "2) Force kill Docker engine"
        echo "3) Check Docker containers"
        echo "4) Check application logs"
        echo "5) Auto-fix (reset + seed + start movement)"
        echo "6) Back to main menu"
EOF

# Create updated case statement
cat > /tmp/updated_case.sh << 'EOF'
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
                kill_docker_engine
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
                print_error "Invalid choice. Please try again."
                sleep 2
                ;;
        esac
EOF

# Replace the menu section
head -n $((start_line - 1)) scripts/fleet_sustainability.sh > /tmp/script_part1.sh
cat /tmp/updated_menu.sh >> /tmp/script_part1.sh

# Find the case statement start
case_start=$(grep -n "case \$choice in" scripts/fleet_sustainability.sh | cut -d: -f1)
if [ -n "$case_start" ]; then
    # Get everything between menu and case
    sed -n "$((end_line + 1)),$((case_start - 1))p" scripts/fleet_sustainability.sh >> /tmp/script_part1.sh
    cat /tmp/updated_case.sh >> /tmp/script_part1.sh
    # Get everything after the case statement
    tail -n +$((case_start + 50)) scripts/fleet_sustainability.sh >> /tmp/script_part1.sh
else
    echo "❌ Could not find case statement"
    exit 1
fi

# Replace the original file
mv /tmp/script_part1.sh scripts/fleet_sustainability.sh

echo "✅ Successfully updated troubleshooting menu with Docker killing option"

# Clean up temp files
rm -f /tmp/updated_menu.sh /tmp/updated_case.sh
