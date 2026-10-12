@ssacli
Feature: ssacli disk failure and replacement
  The ssacli counterpart of megaraid-disk-failure: an HPE Smart Array
  P816i-a with HDDs, each in its own RAID0 array, as ARTESCA deploys them.
  Drive 1I:1:4 (array C, logical drive 3) fails, is replaced, then gets a new
  array (ARTESCA-17960).

  The capture is real ssacli output of a lab node while all its drives were
  healthy, kept to 4 drives, with serials, WWIDs and identifiers replaced by
  fake ones. ssacli getters report no permanent path.

  Background:
    Given the controller captured as "ssacli-4hdd"

  Scenario: All drives are healthy
    Then the drives are reported as:
      | slot   | status | serial   | devicePath | permanentPath |
      | 1I:1:3 | Used   | SIMH0001 | /dev/sdd   |               |
      | 1I:1:4 | Used   | SIMH0002 | /dev/sde   |               |
      | 2I:2:1 | Used   | SIMH0003 | /dev/sdf   |               |
      | 2I:2:2 | Used   | SIMH0004 | /dev/sdg   |               |

  Scenario: A failed drive does not take the paths of the others
    When drive 1I:1:4 fails
    Then the drives are reported as:
      | slot   | status | serial   | devicePath | permanentPath |
      | 1I:1:3 | Used   | SIMH0001 | /dev/sdd   |               |
      | 1I:1:4 | Failed | SIMH0002 |            |               |
      | 2I:2:1 | Used   | SIMH0003 | /dev/sdf   |               |
      | 2I:2:2 | Used   | SIMH0004 | /dev/sdg   |               |

  Scenario: A replacement drive is reported as a new, available drive
    Given drive 1I:1:4 has failed
    When drive 1I:1:4 is replaced by a drive with serial "SIMH0013" and WWN "5000C50030000013"
    Then the drives are reported as:
      | slot   | status         | serial   | devicePath | permanentPath |
      | 1I:1:3 | Used           | SIMH0001 | /dev/sdd   |               |
      | 1I:1:4 | UnassignedGood | SIMH0013 |            |               |
      | 2I:2:1 | Used           | SIMH0003 | /dev/sdf   |               |
      | 2I:2:2 | Used           | SIMH0004 | /dev/sdg   |               |

  Scenario: A replacement drive gets the paths of its new array
    Given drive 1I:1:4 has failed
    And drive 1I:1:4 has been replaced by a drive with serial "SIMH0013" and WWN "5000C50030000013"
    When a volume is created on drive 1I:1:4, exposed as "/dev/sdh" with WWN "600508B1000000000000000000001003"
    Then the drives are reported as:
      | slot   | status | serial   | devicePath | permanentPath |
      | 1I:1:3 | Used   | SIMH0001 | /dev/sdd   |               |
      | 1I:1:4 | Used   | SIMH0013 | /dev/sdh   |               |
      | 2I:2:1 | Used   | SIMH0003 | /dev/sdf   |               |
      | 2I:2:2 | Used   | SIMH0004 | /dev/sdg   |               |
