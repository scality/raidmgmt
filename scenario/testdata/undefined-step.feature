@fake
Feature: undefined step

  Background:
    Given the controller captured as "fake"

  Scenario: x
    When drive 251:1 burns
    Then the drives are reported as:
      | slot  | status | serial | devicePath | permanentPath |
      | 251:1 | Failed | A      |            |               |
