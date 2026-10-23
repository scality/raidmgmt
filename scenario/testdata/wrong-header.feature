@fake
Feature: wrong header

  Background:
    Given the controller captured as "fake"

  Scenario: x
    Then the drives are reported as:
      | slot  | status |
      | 251:1 | Failed |
