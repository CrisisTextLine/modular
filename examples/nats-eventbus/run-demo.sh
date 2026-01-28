#!/bin/bash

# NATS EventBus Demo Runner
# This script helps run the NATS EventBus demonstration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Detect docker compose command
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
elif command -v docker &> /dev/null && docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    echo -e "${RED}❌ Docker Compose not found${NC}"
    echo "Please install Docker and Docker Compose"
    exit 1
fi

# Function to check if NATS is ready
wait_for_nats() {
    echo -e "${BLUE}⏳ Waiting for NATS to be ready...${NC}"
    max_attempts=30
    attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -sf http://localhost:8222/healthz > /dev/null 2>&1; then
            echo -e "${GREEN}✅ NATS is ready!${NC}"
            return 0
        fi
        attempt=$((attempt + 1))
        echo -n "."
        sleep 1
    done
    
    echo -e "${RED}❌ NATS failed to start within timeout${NC}"
    return 1
}

# Function to start services
start_services() {
    echo -e "${BLUE}🚀 Starting NATS service...${NC}"
    $DOCKER_COMPOSE up -d
    
    if wait_for_nats; then
        echo -e "${GREEN}✅ NATS service started successfully${NC}"
        echo -e "${BLUE}📊 NATS is available at:${NC}"
        echo "   - Client port: localhost:4222"
        echo "   - Monitoring:  http://localhost:8222"
    else
        echo -e "${RED}❌ Failed to start NATS service${NC}"
        exit 1
    fi
}

# Function to stop services
stop_services() {
    echo -e "${YELLOW}🛑 Stopping NATS service...${NC}"
    $DOCKER_COMPOSE down
    echo -e "${GREEN}✅ Services stopped${NC}"
}

# Function to show logs
show_logs() {
    echo -e "${BLUE}📋 NATS service logs:${NC}"
    $DOCKER_COMPOSE logs -f
}

# Function to check status
check_status() {
    echo -e "${BLUE}🔍 Service status:${NC}"
    $DOCKER_COMPOSE ps
    
    echo ""
    if curl -sf http://localhost:8222/healthz > /dev/null 2>&1; then
        echo -e "${GREEN}✅ NATS is healthy${NC}"
    else
        echo -e "${RED}❌ NATS is not responding${NC}"
    fi
}

# Function to run the demo with NATS
run_with_nats() {
    echo -e "${BLUE}🚀 Starting full demo with NATS...${NC}"
    
    # Start NATS if not running
    if ! curl -sf http://localhost:8222/healthz > /dev/null 2>&1; then
        start_services
    else
        echo -e "${GREEN}✅ NATS is already running${NC}"
    fi
    
    # Build the application
    echo -e "${BLUE}🔨 Building application...${NC}"
    GOWORK=off go build -o nats-demo .
    
    # Run the application
    echo -e "${GREEN}✅ Starting NATS EventBus Demo...${NC}"
    echo -e "${BLUE}📊 Application output (will run for 15 seconds):${NC}"
    echo ""
    
    # Run the demo for 15 seconds to allow proper pub/sub validation
    # Send SIGINT for graceful shutdown, then SIGKILL if it doesn't respond
    # Temporarily disable exit-on-error to capture timeout exit code
    set +e
    timeout -s INT -k 10s 15s ./nats-demo
    EXIT_CODE=$?
    set -e

    # timeout returns various exit codes:
    # - 0: Process exited cleanly before timeout
    # - 124: Process was terminated by timeout with default signal
    # - 130: Process was terminated by SIGINT (128 + 2)
    # - 137: Process was killed by SIGKILL (128 + 9)
    # - 143: Process was terminated by SIGTERM (128 + 15)
    # We accept these as success (expected termination paths)
    case $EXIT_CODE in
        0|124|130|137|143)
            echo ""
            echo -e "${GREEN}✅ Demo completed successfully${NC}"
            return 0
            ;;
        *)
            echo ""
            echo -e "${RED}❌ Demo failed with exit code $EXIT_CODE${NC}"
            return 1
            ;;
    esac
}

# Function to cleanup everything
cleanup() {
    echo -e "${YELLOW}🧹 Cleaning up...${NC}"
    stop_services
    rm -f nats-demo
    echo -e "${GREEN}✅ Cleanup complete${NC}"
}

# Main script
case "${1:-help}" in
    start)
        start_services
        ;;
    stop)
        stop_services
        ;;
    restart)
        stop_services
        sleep 2
        start_services
        ;;
    logs)
        show_logs
        ;;
    status)
        check_status
        ;;
    run)
        run_with_nats
        ;;
    cleanup)
        cleanup
        ;;
    help|*)
        echo "NATS EventBus Demo Runner"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  start    - Start NATS service"
        echo "  stop     - Stop NATS service"
        echo "  restart  - Restart NATS service"
        echo "  logs     - Show NATS logs"
        echo "  status   - Check service status"
        echo "  run      - Start services and run the demo"
        echo "  cleanup  - Stop services and clean up"
        echo "  help     - Show this help message"
        echo ""
        echo "Examples:"
        echo "  $0 run                    # Run the full demo"
        echo "  $0 start && go run .      # Start NATS and run manually"
        echo "  $0 cleanup                # Stop everything and clean up"
        ;;
esac
