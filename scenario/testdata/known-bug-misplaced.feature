@fake
Feature: known bug misplaced

  Background:
    Given the controller captured as "fake"

  Scenario: x
    When drive 251:1 fails
    But known bug BUG-1 reports instead:
      | slot  | status | serial | devicePath | permanentPath |
      | 251:1 | Failed | A      |            |               |
    Then the drives are reported as:
      | slot  | status | serial | devicePath | permanentPath |
      | 251:1 | Failed | A      |            |               |
