#!/bin/bash
# Test script for validating Go module version handling
# This demonstrates the logic added to the release workflows

set -euo pipefail

echo "=== Go Module Version Handling Test ==="
echo

# Function to extract major version from a version string
extract_major_version() {
    local version="$1"
    local major="${version#v}"
    major="${major%%.*}"
    echo "$major"
}

# Function to determine correct module path for a version
get_module_path() {
    local base_path="$1"
    local version="$2"
    local major
    major=$(extract_major_version "$version")
    
    if [ "$major" -ge 2 ]; then
        echo "${base_path}/v${major}"
    else
        echo "${base_path}"
    fi
}

# Test cases
test_version() {
    local version="$1"
    local base_path="github.com/CrisisTextLine/modular/modules/reverseproxy"
    local expected_path
    expected_path=$(get_module_path "$base_path" "$version")
    
    echo "Version: $version"
    echo "  Major: $(extract_major_version "$version")"
    echo "  Module Path: $expected_path"
    echo
}

echo "--- Test Cases ---"
test_version "v0.1.0"
test_version "v1.0.0"
test_version "v1.5.2"
test_version "v2.0.0"
test_version "v2.1.0"
test_version "v3.0.0"

echo "=== Core Framework Test ==="
echo
base_path="github.com/CrisisTextLine/modular"
for version in "v1.0.0" "v2.0.0" "v3.0.0"; do
    echo "Version: $version -> $(get_module_path "$base_path" "$version")"
done

echo
echo "✓ All test cases executed successfully"
