Feature: Feature-flagged composite route with dry-run fallback
  As a system administrator
  I want feature flags to control composite routes with dry-run comparison
  So that I can safely test new routing configurations while maintaining fallback behavior

  Background:
    Given I have a modular application with reverse proxy module configured

  Scenario: Feature-flagged composite route with dry-run fallback
    Given I have a composite route guarded by feature flag
    When I enable module-level dry run mode
    And I disable the feature flag for composite route
    And I make a request to the composite route
    Then the response should come from the alternative backend
    And dry-run handler should compare alternative with primary
    And log output should include comparison diffs
    And CloudEvents should show request.received and request.failed when backends diverge