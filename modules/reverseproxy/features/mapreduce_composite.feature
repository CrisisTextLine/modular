Feature: Map/Reduce Composite Routes
  As a system architect
  I want to use map/reduce composite routes
  So that I can aggregate data from multiple backends intelligently

  Background:
    Given a reverse proxy module with map/reduce support

  Scenario: Sequential map/reduce with conversation list and follow-ups
    Given a backend "conversations" that returns a list of conversations
    And a backend "followups" that accepts conversation IDs and returns follow-up data
    And a sequential map/reduce route configured to:
      | extract_path           | conversations      |
      | extract_field          | id                 |
      | target_request_field   | conversation_ids   |
      | merge_strategy         | enrich             |
      | merge_into_field       | followup_data      |
    When I make a GET request to the map/reduce route
    Then the response status code should be 200
    And the response should contain the original conversation list
    And the response should contain enriched followup data
    And each conversation should have its follow-up information if available

  Scenario: Sequential map/reduce with empty source list
    Given a backend "source" that returns an empty list
    And a backend "target" that would process IDs
    And a sequential map/reduce route configured with allow_empty_responses true
    When I make a GET request to the map/reduce route
    Then the response status code should be 200
    And the response should be the source response unchanged
    And the target backend should not have been called

  Scenario: Sequential map/reduce without allow_empty_responses
    Given a backend "source" that returns an empty list
    And a backend "target" that would process IDs
    And a sequential map/reduce route configured with allow_empty_responses false
    When I make a GET request to the map/reduce route
    Then the response status code should be 204
    And the target backend should not have been called

  Scenario: Parallel map/reduce with join on common field
    Given a backend "base" that returns items with IDs
    And a backend "ancillary" that returns additional data for some IDs
    And a parallel map/reduce route configured to:
      | join_field             | id                 |
      | merge_strategy         | join               |
      | merge_into_field       | extra_data         |
    When I make a GET request to the map/reduce route
    Then the response status code should be 200
    And the response should be an array
    And each item should have data from both backends joined by ID
    And items without ancillary data should still be present

  Scenario: Parallel map/reduce with filtering
    Given a backend "base" that returns 5 items
    And a backend "ancillary" that returns data for only 3 items
    And a parallel map/reduce route configured with:
      | join_field             | id                 |
      | merge_strategy         | join               |
      | filter_on_empty        | true               |
    When I make a GET request to the map/reduce route
    Then the response status code should be 200
    And the response should contain exactly 3 items
    And all items should have ancillary data present

  Scenario: Sequential map/reduce with nested data extraction
    Given a backend "source" that returns nested data structure
    And the source has items at path "data.items"
    And a backend "target" that processes extracted IDs
    And a sequential map/reduce route configured to:
      | extract_path           | data.items         |
      | extract_field          | item_id            |
      | target_request_field   | ids                |
      | merge_strategy         | enrich             |
    When I make a GET request to the map/reduce route
    Then the response status code should be 200
    And the nested structure should be preserved in the response
    And the enrichment should be added to the response

  Scenario: Map/reduce with complex objects and multiple fields
    Given a backend "conversations" that returns complex conversation objects
    And each conversation has id, title, status, and metadata
    And a backend "participants" that returns participant info for conversations
    And a parallel map/reduce route configured to join on "conversation_id"
    When I make a GET request to the map/reduce route
    Then the response status code should be 200
    And each conversation should have its original fields
    And each conversation should have participant data merged in

  Scenario: Sequential map/reduce with source backend error
    Given a backend "source" that returns status 500
    And a backend "target" that is healthy
    And a sequential map/reduce route
    When I make a GET request to the map/reduce route
    Then the response status code should be 502
    And the error message should indicate source backend failure
    And the target backend should not have been called

  Scenario: Sequential map/reduce with target backend error and allow_empty
    Given a backend "source" that returns valid data
    And a backend "target" that returns status 500
    And a sequential map/reduce route configured with allow_empty_responses true
    When I make a GET request to the map/reduce route
    Then the response status code should be 200
    And the response should be the source response unchanged

  Scenario: Sequential map/reduce with target backend error without allow_empty
    Given a backend "source" that returns valid data
    And a backend "target" that returns status 500
    And a sequential map/reduce route configured with allow_empty_responses false
    When I make a GET request to the map/reduce route
    Then the response status code should be 502
    And the error message should indicate target backend failure

  Scenario: Parallel map/reduce with all backends failing
    Given a backend "base" that returns status 500
    And a backend "ancillary" that returns status 503
    And a parallel map/reduce route configured with allow_empty_responses false
    When I make a GET request to the map/reduce route
    Then the response status code should be 502
    And the error message should indicate no successful responses

  Scenario: Parallel map/reduce with partial backend failures
    Given a backend "base" that returns valid data
    And a backend "ancillary" that returns status 500
    And a backend "extra" that returns valid data
    And a parallel map/reduce route configured with allow_empty_responses true
    When I make a GET request to the map/reduce route
    Then the response status code should be 200
    And the response should contain data from successful backends only

  Scenario: Map/reduce with flat merge strategy
    Given a backend "user_service" that returns user profile data
    And a backend "analytics_service" that returns user analytics
    And a sequential map/reduce route configured with merge_strategy "flat"
    When I make a GET request to the map/reduce route
    Then the response status code should be 200
    And all fields from both backends should be at the top level
    And there should be no nested backend keys

  Scenario: Map/reduce with nested merge strategy
    Given a backend "service_a" that returns data A
    And a backend "service_b" that returns data B
    And a sequential map/reduce route configured with merge_strategy "nested"
    When I make a GET request to the map/reduce route
    Then the response status code should be 200
    And the response should have a "service_a" field with data A
    And the response should have a "service_b" field with data B

  Scenario: Map/reduce preserving empty arrays vs null
    Given a backend "source" that returns items with some null fields
    And a backend "target" that adds data
    And a sequential map/reduce route
    When I make a GET request to the map/reduce route
    Then the response status code should be 200
    And null fields should remain null
    And empty arrays should remain empty arrays

  Scenario: Map/reduce with custom HTTP method for target request
    Given a backend "source" that returns a list
    And a backend "target" that expects a PUT request
    And a sequential map/reduce route configured with:
      | target_request_method  | PUT                |
      | target_request_path    | /bulk/update       |
    When I make a GET request to the map/reduce route
    Then the target backend should receive a PUT request to "/bulk/update"
    And the response status code should be 200

  Scenario: Large dataset map/reduce performance
    Given a backend "source" that returns 1000 items
    And a backend "target" that processes IDs
    And a sequential map/reduce route
    When I make a GET request to the map/reduce route
    Then the response status code should be 200
    And the response should contain all 1000 items enriched
    And the request should complete in a reasonable time

  Scenario: Parallel map/reduce with deterministic ordering
    Given multiple backends that return data in random order
    And a parallel map/reduce route with join strategy
    When I make multiple requests to the map/reduce route
    Then all responses should have consistent ordering based on the first backend
    And the join logic should be deterministic
