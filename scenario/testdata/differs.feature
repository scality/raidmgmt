@fake
Feature: differs

  Background:
    Given the controller captured as "fake"

  Scenario: x
    Then the drives are reported as:
      | slot  | status | serial | devicePath | permanentPath |
      | 251:1 | Used   | A      |            |               |
