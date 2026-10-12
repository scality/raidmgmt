@storcli2
Feature: storcli2 disk failure and replacement
  The storcli2 counterpart of megaraid-disk-failure: a controller driven by
  storcli2 with HDDs in enclosure 306, each in its own RAID0 volume, as
  ARTESCA deploys them. Drive 306:2 fails, is replaced, then gets a new
  volume (ARTESCA-17960).

  The capture is real storcli2 output of a lab node while all its drives
  were healthy, kept to its first 4 drives, with serials, WWNs and volume
  identifiers replaced by fake ones.

  Background:
    Given the controller captured as "storcli2-4hdd"

  Scenario: All drives are healthy
    Then the drives are reported as:
      | slot  | status | serial               | devicePath | permanentPath                                          |
      | 306:0 | Used   | SIMD2001000000000000 | /dev/sdb   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000001 |
      | 306:1 | Used   | SIMD2002000000000000 | /dev/sdc   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000002 |
      | 306:2 | Used   | SIMD2003000000000000 | /dev/sdd   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000003 |
      | 306:3 | Used   | SIMD2004000000000000 | /dev/sde   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000004 |

  Scenario: A failed drive does not take the paths of the others
    When drive 306:2 fails
    Then the drives are reported as:
      | slot  | status | serial               | devicePath | permanentPath                                          |
      | 306:0 | Used   | SIMD2001000000000000 | /dev/sdb   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000001 |
      | 306:1 | Used   | SIMD2002000000000000 | /dev/sdc   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000002 |
      | 306:2 | Failed | SIMD2003000000000000 |            |                                                        |
      | 306:3 | Used   | SIMD2004000000000000 | /dev/sde   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000004 |
    # The permanent path of the offline volume is still built from its NAA
    # Id, although the volume has no block device and no by-id link.
    But known bug ARTESCA-17960 reports instead:
      | slot  | status | serial               | devicePath | permanentPath                                          |
      | 306:0 | Used   | SIMD2001000000000000 | /dev/sdb   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000001 |
      | 306:1 | Used   | SIMD2002000000000000 | /dev/sdc   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000002 |
      | 306:2 | Failed | SIMD2003000000000000 |            | /dev/disk/by-id/wwn-0x600062b2000000000000000000000003 |
      | 306:3 | Used   | SIMD2004000000000000 | /dev/sde   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000004 |

  Scenario: A replacement drive is reported as a new, available drive
    Given drive 306:2 has failed
    When drive 306:2 is replaced by a drive with serial "SIMD2013000000000000" and WWN "5000C50020000013"
    Then the drives are reported as:
      | slot  | status         | serial               | devicePath | permanentPath                                          |
      | 306:0 | Used           | SIMD2001000000000000 | /dev/sdb   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000001 |
      | 306:1 | Used           | SIMD2002000000000000 | /dev/sdc   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000002 |
      | 306:2 | UnassignedGood | SIMD2013000000000000 |            |                                                        |
      | 306:3 | Used           | SIMD2004000000000000 | /dev/sde   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000004 |

  Scenario: A replacement drive gets the paths of its new volume
    Given drive 306:2 has failed
    And drive 306:2 has been replaced by a drive with serial "SIMD2013000000000000" and WWN "5000C50020000013"
    When a volume is created on drive 306:2, exposed as "/dev/sdf" with WWN "600062b2000000000000000000001003"
    Then the drives are reported as:
      | slot  | status | serial               | devicePath | permanentPath                                          |
      | 306:0 | Used   | SIMD2001000000000000 | /dev/sdb   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000001 |
      | 306:1 | Used   | SIMD2002000000000000 | /dev/sdc   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000002 |
      | 306:2 | Used   | SIMD2013000000000000 | /dev/sdf   | /dev/disk/by-id/wwn-0x600062b2000000000000000000001003 |
      | 306:3 | Used   | SIMD2004000000000000 | /dev/sde   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000004 |
