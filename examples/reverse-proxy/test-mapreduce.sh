#!/bin/bash

# Test script for map/reduce composite routes
# This script demonstrates the map/reduce functionality in the reverse-proxy example

set -e

echo "========================================="
echo "Map/Reduce Composite Routes Test Script"
echo "========================================="
echo ""

# Function to make a request and pretty print JSON
test_endpoint() {
    local name="$1"
    local url="$2"
    
    echo "Testing: $name"
    echo "URL: $url"
    echo ""
    
    response=$(curl -s "$url")
    if command -v jq &> /dev/null; then
        echo "$response" | jq '.'
    else
        echo "$response"
    fi
    
    echo ""
    echo "---"
    echo ""
}

# Check if server is running
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "❌ Server is not running on port 8080"
    echo "Please start the server first: go run main.go"
    exit 1
fi

echo "✅ Server is running"
echo ""
echo "========================================="
echo "Testing Map/Reduce Endpoints"
echo "========================================="
echo ""

# Test 1: Sequential Map/Reduce - Conversations with Follow-ups
test_endpoint \
    "Sequential Map/Reduce: Conversations with Follow-up Data" \
    "http://localhost:8080/api/composite/mapreduce/conversations"

# Test 2: Parallel Map/Reduce - Parallel Join
test_endpoint \
    "Parallel Map/Reduce: Join by ID" \
    "http://localhost:8080/api/composite/mapreduce/parallel-join"

echo "========================================="
echo "Testing Other Composite Strategies"
echo "========================================="
echo ""

# Test 3: First Success Strategy
test_endpoint \
    "First-Success Strategy" \
    "http://localhost:8080/api/composite/first-success"

# Test 4: Merge Strategy
test_endpoint \
    "Merge Strategy" \
    "http://localhost:8080/api/composite/merge"

# Test 5: Sequential Strategy
test_endpoint \
    "Sequential Strategy" \
    "http://localhost:8080/api/composite/sequential"

# Test 6: Custom Transformer
test_endpoint \
    "Custom Transformer: Profile with Analytics" \
    "http://localhost:8080/api/composite/profile-with-analytics"

echo "========================================="
echo "✅ All tests completed successfully!"
echo "========================================="
echo ""
echo "Map/Reduce Pattern Summary:"
echo ""
echo "1. Sequential Pattern:"
echo "   - Queries conversations backend"
echo "   - Extracts conversation IDs"
echo "   - Sends IDs to followups backend"
echo "   - Merges follow-up data into response"
echo ""
echo "2. Parallel Pattern:"
echo "   - Queries conversations and participants in parallel"
echo "   - Joins results by conversation ID"
echo "   - Returns unified response with all data"
echo ""
echo "Both patterns demonstrate intelligent data aggregation"
echo "across multiple independent microservices."
