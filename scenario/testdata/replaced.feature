@fake
Feature: replaced

  Background:
    Given the controller captured as "fake"

  Scenario: a drive is replaced
    When drive 251:1 is replaced by a drive with serial "B" and WWN "W"
    Then the drives are reported as:
      | slot  | status | serial | devicePath | permanentPath |
      | 251:1 | Failed | A      |            |               |

  Scenario: a drive is replaced by a used drive, exposed to the OS
    When drive 251:1 is replaced by a used drive with serial "B" and WWN "W", exposed as "/dev/sdi"
    Then the drives are reported as:
      | slot  | status | serial | devicePath | permanentPath |
      | 251:1 | Failed | A      |            |               |
