# Hardware scenarios

Replay what happens on a RAID controller, such as a drive failing and being
replaced, from real controller output, and check what raidmgmt and its
consumers report, without the hardware.

A scenario is a YAML file in [`scenarios/`](scenarios): a capture of a real
controller, then phases. Each phase changes drives and volumes and gives, as a
table, the disks expected:

```yaml
- name: failed
  description: 251:3 fails, its RAID0 volume goes offline.
  knownBug:
    ticket: ARTESCA-17960
    reported: |
      slot    status  serial                devicePath  permanentPath
      251:1   Used    SIMD0001000000000000  -           -
      251:3   Failed  SIMD0003              -           -
  changes:
    drives:
      "251:3": {state: Failed, serial: SIMD0003}
    volumes:
      "237": {state: OfLn, exposed: false}
  expect: |
    slot    status  serial                devicePath  permanentPath
    251:1   Used    SIMD0001000000000000  /dev/sdl    /dev/disk/by-id/wwn-0x6000…
    251:3   Failed  SIMD0003              -           -
```

- Changes are cumulative: a phase starts from the state of the previous one.
- States use the controller's vocabulary (`Onln`, `Failed`, `UGood`, `OfLn`
  for MegaRAID).
- The table is what must be reported per drive: status, serial, and the
  device and permanent paths of the volume holding it. `-` means empty.
- `knownBug` documents a bug and the table reported today because of it.
  Tests pass while that table is reported. They fail when the expected table
  is reported (the bug is fixed: remove the `knownBug` block) and when neither
  is (the behaviour changed).

## Where scenarios are played

- `megaraidsim/adapter_test.go` plays every MegaRAID scenario (`controller:
  megaraid`, storcli64) through the raidmgmt megaraid adapter of this
  repository.
- `storcli2sim/adapter_test.go` plays every storcli2 scenario (`controller:
  storcli2`) through the raidmgmt storcli2 adapter.
- `ssaclisim/adapter_test.go` plays every ssacli scenario (`controller:
  ssacli`, HPE Smart Array) through the raidmgmt ssacli getters.
- Consumers import this module to play the same scenarios through their own
  code. scality/disk-management-agent checks its DiscoveredPhysicalDisk
  status, which the Storage Service UI reads.

This is a separate module (`github.com/scality/raidmgmt/scenario`) so that
consumers can use it without changing the raidmgmt version they test.

## Adding a scenario

Write a YAML file in `scenarios/` on an existing capture and run
`go test ./...` in this directory.

## Adding a controller

1. Capture its CLI output on a lab machine, one file per command the adapter
   runs, in `<backend>/captures/<name>/`.
2. Write a backend that answers those commands from the capture with the
   changes of a phase applied, as [`megaraidsim`](megaraidsim) does for
   storcli64, and gives the host files the adapter checks.
3. Add a test playing the scenarios of that controller through its adapter;
   `scenario.Report` turns what an adapter reports into the disks of a table.

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
- `ssaclisim/captures/ssacli-4hdd`: HPE Smart Array P816i-a, SAS HDDs each
  in its own RAID0 array. Captured with `ssacli controller slot=0 ... show
  detail` and `show config`, kept to 4 drives (1I:1:3, 1I:1:4, 2I:2:1,
  2I:2:2, logical drives 2-5), with the same fake identifiers (serials,
  WWIDs, unique identifiers, controller and host serials). ssacli prints
  text: the backend edits the lines of the captured sections.
