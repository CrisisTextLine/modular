package main

import "errors"

var (
errLoadTestFailed = errors.New("load test failed: success rate below 80%")
errUnstableBackendNotFound = errors.New("unstable backend not found for failover testing")
)
