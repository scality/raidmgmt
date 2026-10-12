# Hardware scenarios

Replay what happens on a RAID controller, such as a drive failing and being
replaced, from real controller output, and check what raidmgmt reports,
without the hardware.

Scenarios are Gherkin features in [`features/`](features), one per
controller family and story, tagged with the family (`@megaraid`,
`@storcli2`). The `Background` names the capture of a real controller;
each `Scenario` plays events on its drives and gives, as a table, the disks
expected:

```gherkin
@megaraid
Feature: MegaRAID disk failure and replacement

  Background:
    Given the controller captured as "megaraid9560-4hdd"

  Scenario: A failed drive does not take the paths of the others
    When drive 251:3 fails
    Then the drives are reported as:
      | slot  | status | serial               | devicePath | permanentPath               |
      | 251:1 | Used   | SIMD0001000000000000 | /dev/sdl   | /dev/disk/by-id/wwn-0x6000… |
      | 251:3 | Failed | SIMD0003             |            |                             |
    But known bug ARTESCA-17960 reports instead:
      | slot  | status | serial               | devicePath | permanentPath |
      | 251:1 | Used   | SIMD0001000000000000 |            |               |
      | 251:3 | Failed | SIMD0003             |            |               |
```

## Steps

| Step | Event |
|---|---|
| `Given the controller captured as "<capture>"` | Start from a capture |
| `drive <slot> fails` / `has failed` | The drive fails; its volume goes offline, no longer a block device |
| `drive <slot> is replaced` / `has been replaced by a drive with serial "<serial>" and WWN "<wwn>"` | Its volume is deleted and a new drive, in no volume, takes the slot |
| … `by a used drive with …` | The new drive already carries data, such as a filesystem |
| … `, exposed as "<device>"` | The controller exposes the new, free drive to the OS as that block device, as an HPE Smart Array in Mixed mode does |
| `a volume is created` / `has been created on drive <slot>, exposed as "<device>" with WWN "<wwn>"` | The volume the drive had in the capture is created again on it |
| `Then the drives are reported as:` | Check the table: status, serial, and the device and permanent paths of each drive, merged as disk-management-agent does (its own paths, the empty ones filled from the volume holding it). An empty cell is an empty value |
| `But known bug <ticket> reports instead:` | Right after a `Then`: the table reported today because of the bug |

- Each scenario stands alone: it replays its own events on the capture, the
  ones it depends on written as `Given` steps.
- Events are written the same for every controller; each backend applies
  them as its controller shows them (states, serials, drive groups), as
  documented in its package. A backend rejects events in an order that
  cannot happen: only a drive in a volume fails, only a failed drive is
  replaced, and a volume is created only once the old one is deleted.
- A known bug passes while its table is reported. It fails when the
  expected table is reported (the bug is fixed: remove the step) and when
  neither is (the behaviour changed).
- Steps are run by [godog](https://github.com/cucumber/godog), strictly: a
  step with no definition fails. When a table differs, the error prints the
  table reported, ready to paste.

## Where scenarios are played

- `megaraidsim/adapter_test.go` plays every `@megaraid` scenario (storcli64)
  through the raidmgmt megaraid adapter of this repository, with
  `scenario.Run`.
- `storcli2sim/adapter_test.go` plays every `@storcli2` scenario through the
  raidmgmt storcli2 adapter.
- `steps_internal_test.go` checks the steps themselves on the features of
  [`testdata/`](testdata), in particular that they fail when they should.

This is a separate module (`github.com/scality/raidmgmt/scenario`), which
uses the raidmgmt code of this repository through a `replace`: godog and
the captures stay out of the dependencies and the downloads of raidmgmt.

## Adding a scenario

Add a `Scenario` to a feature, or a `.feature` file in `features/` tagged
with its controller family, on an existing capture, and run `go test ./...`
in this directory. A new kind of event needs a step in `steps.go` and its
handling in every backend.

## Adding a controller

1. Capture its CLI output on a lab machine, one file per command the adapter
   runs, in `<backend>/captures/<name>/`.
2. Write a backend that answers those commands from the capture with the
   events applied, as [`megaraidsim`](megaraidsim) does for storcli64, and
   gives the host files the adapter checks.
3. Add a test calling `scenario.Run` with the tag of that controller and a
   player reporting through its adapter; `scenario.Report` turns what an
   adapter reports into the disks of a table.

## Captures

- `megaraidsim/captures/megaraid9560-4hdd`: MegaRAID 9560-8i, SEAGATE
  ST18000NM000D drives in enclosure 251, one RAID0 volume each (storcli64
  007.1616). Captured with `storcli64 <selector> show all J` on 12 drives,
  kept to the first 4 (251:1-4, volumes 239-236) to keep scenarios short.
  Serials, WWNs and controller identifiers are replaced by fake ones, also in
  the hex inquiry data; keep it so when adding a capture.
- `storcli2sim/captures/storcli2-4hdd`: storcli2 controller, SEAGATE
  ST10000NM018B drives in enclosure 306, one RAID0 volume each (storcli2
  008.0005). Captured with `storcli2 <selector> show all J` on 24 drives,
  kept to the first 4 (306:0-3, volumes 1-4), with the same fake identifiers
  (drive serials, WWNs and SAS addresses, volume NAA Ids and serials).
