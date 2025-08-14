Feature: Reverse Proxy Module
  As a developer using the Modular framework
  I want to use the reverse proxy module for load balancing and request routing
  So that I can distribute traffic across multiple backend services

  Background:
    Given I have a modular application with reverse proxy module configured

  Scenario: Reverse proxy module initialization
    When the reverse proxy module is initialized
    Then the proxy service should be available
    And the module should be ready to route requests

  Scenario: Single backend proxy routing
    Given I have a reverse proxy configured with a single backend
    When I send a request to the proxy
    Then the request should be forwarded to the backend
    And the response should be returned to the client

  Scenario: Multiple backend load balancing
    Given I have a reverse proxy configured with multiple backends
    When I send multiple requests to the proxy
    Then requests should be distributed across all backends
    And load balancing should be applied

  Scenario: Backend health checking
    Given I have a reverse proxy with health checks enabled
    When a backend becomes unavailable
    Then the proxy should detect the failure
    And route traffic only to healthy backends

  Scenario: Circuit breaker functionality
    Given I have a reverse proxy with circuit breaker enabled
    When a backend fails repeatedly
    Then the circuit breaker should open
    And requests should be handled gracefully

  Scenario: Response caching
    Given I have a reverse proxy with caching enabled
    When I send the same request multiple times
    Then the first request should hit the backend
    And subsequent requests should be served from cache

  Scenario: Tenant-aware routing
    Given I have a tenant-aware reverse proxy configured
    When I send requests with different tenant contexts
    Then requests should be routed based on tenant configuration
    And tenant isolation should be maintained

  Scenario: Composite response handling
    Given I have a reverse proxy configured for composite responses
    When I send a request that requires multiple backend calls
    Then the proxy should call all required backends
    And combine the responses into a single response

  Scenario: Request transformation
    Given I have a reverse proxy with request transformation configured
    When I send a request to the proxy
    Then the request should be transformed before forwarding
    And the backend should receive the transformed request

  Scenario: Graceful shutdown
    Given I have an active reverse proxy with ongoing requests
    When the module is stopped
    Then ongoing requests should be completed
    And new requests should be rejected gracefully

  Scenario: Emit events during proxy lifecycle
    Given I have a reverse proxy with event observation enabled
    When the reverse proxy module starts
    Then a proxy created event should be emitted
    And a proxy started event should be emitted
    And a module started event should be emitted
    And the events should contain proxy configuration details
    When the reverse proxy module stops
    Then a proxy stopped event should be emitted
    And a module stopped event should be emitted

  Scenario: Emit events during request routing
    Given I have a reverse proxy with event observation enabled
    And I have a backend service configured
    When I send a request to the reverse proxy
    Then a request received event should be emitted
    And the event should contain request details
    When the request is successfully proxied to the backend
    Then a request proxied event should be emitted
    And the event should contain backend and response details

  Scenario: Emit events during request failures
    Given I have a reverse proxy with event observation enabled
    And I have an unavailable backend service configured
    When I send a request to the reverse proxy
    Then a request received event should be emitted
    When the request fails to reach the backend
    Then a request failed event should be emitted
    And the event should contain error details

  Scenario: Emit events during backend health management
    Given I have a reverse proxy with event observation enabled
    And I have backends with health checking enabled
    When a backend becomes healthy
    Then a backend healthy event should be emitted
    And the event should contain backend health details
    When a backend becomes unhealthy
    Then a backend unhealthy event should be emitted
    And the event should contain health failure details

  Scenario: Emit events during backend management
    Given I have a reverse proxy with event observation enabled
    When a new backend is added to the configuration
    Then a backend added event should be emitted
    And the event should contain backend configuration
    When a backend is removed from the configuration
    Then a backend removed event should be emitted
    And the event should contain removal details

  Scenario: Emit events during load balancing decisions
    Given I have a reverse proxy with event observation enabled
    And I have multiple backends configured
    When load balancing decisions are made
    Then load balance decision events should be emitted
    And the events should contain selected backend information
    When round-robin load balancing is used
    Then round-robin events should be emitted
    And the events should contain rotation details

  Scenario: Emit events during circuit breaker operations
    Given I have a reverse proxy with event observation enabled
    And I have circuit breaker enabled for backends
    When a circuit breaker opens due to failures
    Then a circuit breaker open event should be emitted
    And the event should contain failure threshold details
    When a circuit breaker transitions to half-open
    Then a circuit breaker half-open event should be emitted
    When a circuit breaker closes after recovery
    Then a circuit breaker closed event should be emitted