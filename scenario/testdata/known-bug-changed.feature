@fake
Feature: known bug changed

  Background:
    Given the controller captured as "fake"

  Scenario: x
    Then the drives are reported as:
      | slot  | status | serial | devicePath | permanentPath |
      | 251:1 | Used   | A      |            |               |
    But known bug BUG-1 reports instead:
      | slot  | status | serial | devicePath | permanentPath |
      | 251:1 | Failed | B      |            |               |
