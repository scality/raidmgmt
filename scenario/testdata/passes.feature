@fake
Feature: passes

  Background:
    Given the controller captured as "fake"

  Scenario: a drive fails
    When drive 251:1 fails
    Then the drives are reported as:
      | slot  | status | serial | devicePath | permanentPath |
      | 251:1 | Failed | A      |            |               |

  Scenario: nothing happens
    Then the drives are reported as:
      | slot  | status | serial | devicePath | permanentPath |
      | 251:1 | Failed | A      |            |               |
