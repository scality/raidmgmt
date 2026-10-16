@megaraid
Feature: MegaRAID disk failure and replacement
  A MegaRAID 9560-8i with HDDs in enclosure 251, each in its own RAID0
  volume, as ARTESCA deploys them. Drive 251:3 fails, is replaced, then gets a
  new volume (RD-2154, ARTESCA-17960).

  The capture is real storcli output of a lab node while all its drives
  were healthy, kept to its first 4 drives, with serials and WWNs replaced by
  fake ones.

  Each table lists, per drive, what raidmgmt must report: the drive, and
  the paths of the volume holding it.

  Background:
    Given the controller captured as "megaraid9560-4hdd"

  Scenario: All drives are healthy
    Then the drives are reported as:
      | slot  | status | serial               | devicePath | permanentPath                                          |
      | 251:1 | Used   | SIMD0001000000000000 | /dev/sdl   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000239 |
      | 251:2 | Used   | SIMD0002000000000000 | /dev/sdk   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000238 |
      | 251:3 | Used   | SIMD0003000000000000 | /dev/sdj   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000237 |
      | 251:4 | Used   | SIMD0004000000000000 | /dev/sdi   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000236 |

  Scenario: A failed drive does not take the paths of the others
    When drive 251:3 fails
    Then the drives are reported as:
      | slot  | status | serial               | devicePath | permanentPath                                          |
      | 251:1 | Used   | SIMD0001000000000000 | /dev/sdl   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000239 |
      | 251:2 | Used   | SIMD0002000000000000 | /dev/sdk   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000238 |
      | 251:3 | Failed | SIMD0003             |            |                                                        |
      | 251:4 | Used   | SIMD0004000000000000 | /dev/sdi   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000236 |
    # The offline volume has no by-id link: its path cannot be resolved, which
    # fails the listing of all the volumes of the controller.
    But known bug ARTESCA-17960 reports instead:
      | slot  | status | serial               | devicePath | permanentPath |
      | 251:1 | Used   | SIMD0001000000000000 |            |               |
      | 251:2 | Used   | SIMD0002000000000000 |            |               |
      | 251:3 | Failed | SIMD0003             |            |               |
      | 251:4 | Used   | SIMD0004000000000000 |            |               |

  Scenario: A replacement drive is reported as a new, available drive
    Given drive 251:3 has failed
    When drive 251:3 is replaced by a drive with serial "SIMD0013000000000000" and WWN "5000C50000000013"
    Then the drives are reported as:
      | slot  | status         | serial               | devicePath | permanentPath                                          |
      | 251:1 | Used           | SIMD0001000000000000 | /dev/sdl   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000239 |
      | 251:2 | Used           | SIMD0002000000000000 | /dev/sdk   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000238 |
      | 251:3 | UnassignedGood | SIMD0013000000000000 |            |                                                        |
      | 251:4 | Used           | SIMD0004000000000000 | /dev/sdi   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000236 |

  Scenario: A replacement drive gets the paths of its new volume
    Given drive 251:3 has failed
    And drive 251:3 has been replaced by a drive with serial "SIMD0013000000000000" and WWN "5000C50000000013"
    When a volume is created on drive 251:3, exposed as "/dev/sdm" with WWN "600062b2000000000000000000001237"
    Then the drives are reported as:
      | slot  | status | serial               | devicePath | permanentPath                                          |
      | 251:1 | Used   | SIMD0001000000000000 | /dev/sdl   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000239 |
      | 251:2 | Used   | SIMD0002000000000000 | /dev/sdk   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000238 |
      | 251:3 | Used   | SIMD0013000000000000 | /dev/sdm   | /dev/disk/by-id/wwn-0x600062b2000000000000000000001237 |
      | 251:4 | Used   | SIMD0004000000000000 | /dev/sdi   | /dev/disk/by-id/wwn-0x600062b2000000000000000000000236 |
