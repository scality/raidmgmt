@fake
Feature: not reported

  Background:
    Given the controller captured as "fake"

  Scenario: x
    Then the drives are reported as:
      | slot  | status | serial | devicePath | permanentPath |
      | 251:1 | Failed | A      |            |               |
      | 251:2 | Used   | B      |            |               |
