# Testdata for MegaRAID (storcli / storcli2)

This directory contains JSON outputs captured from storcli and storcli2 CLI tools,
used as mock data in unit tests.

## Directory Structure

```
testdata/
├── collect_storcli2_testdata.sh   # Script to collect storcli2 outputs from server
├── README.md                       # This file
├── controllers/
│   ├── storcli/                   # storcli v1 (storcli64) outputs
│   │   ├── all.json
│   │   ├── c0.json
│   │   ├── c0_s12_UGood.json
│   │   └── c5_invalid.json
│   └── storcli2/                  # storcli2 outputs
│       ├── all.json
│       ├── c0.json
│       ├── c0_s12_UGood.json
│       └── c5_invalid.json
├── logicalvolumes/
│   ├── cacheoptions/
│   │   └── success.json           # storcli v1
│   ├── create/
│   │   ├── fail.json
│   │   └── success.json
│   ├── delete/
│   │   ├── fail_invalid.json
│   │   ├── fail_vdNotExist.json
│   │   └── success.json
│   ├── migrate/
│   │   └── fail.json
│   └── show/
│       ├── all.json
│       ├── all_NOv228.json
│       ├── v228.json ... v239.json
│       └── v999_invalid.json
└── physicaldrives/
    ├── blink/
    │   ├── start.json
    │   └── stop.json
    ├── jbod/
    │   ├── disable/
    │   │   └── fail.json
    │   └── enable/
    │       └── fail.json
    └── show/
        ├── all.json
        ├── e251s1.json ... e251s12.json
        ├── e251s12_UGood.json
        └── e251s99_invalid.json
```

## Command → Testdata File Mapping

### storcli v1 (`/opt/MegaRAID/storcli/storcli64`)

| Testdata File | storcli Command | Go Code Path |
|---|---|---|
| `controllers/storcli/all.json` | `storcli64 show all J` | `adapter.showAll()` → `runner.Run(["show", "all"])` |
| `controllers/storcli/c0.json` | `storcli64 /c0 show all J` | `adapter.showAllController("/c0")` → `runner.Run(["/c0", "show", "all"])` |
| `controllers/storcli/c0_s12_UGood.json` | `storcli64 /c0 show all J` *(with slot 12 in UGood state)* | Same as c0.json but pre-VD-creation |
| `controllers/storcli/c5_invalid.json` | `storcli64 /c5 show all J` | Error: controller not found |
| `physicaldrives/show/all.json` | `storcli64 /c0/eall/sall show all J` | All drives detailed info |
| `physicaldrives/show/e251s1.json` | `storcli64 /c0/e251/s1 show all J` | `adapter.showAllPhysicalDrive("/c0/e251/s1")` → `runner.Run(["/c0/e251/s1", "show", "all"])` |
| `physicaldrives/show/e251s12_UGood.json` | `storcli64 /c0/e251/s12 show all J` *(drive in UGood state)* | Drive in unconfigured-good state |
| `physicaldrives/show/e251s99_invalid.json` | `storcli64 /c0/e251/s99 show all J` | Error: drive not found |
| `physicaldrives/blink/start.json` | `storcli64 /c0/e251/s1 start locate J` | `adapter.blink(metadata, "start")` → `runner.Run(["/c0/e251/s1", "start", "locate"])` |
| `physicaldrives/blink/stop.json` | `storcli64 /c0/e251/s1 stop locate J` | `adapter.blink(metadata, "stop")` → `runner.Run(["/c0/e251/s1", "stop", "locate"])` |
| `physicaldrives/jbod/enable/fail.json` | `storcli64 /c0/e251/s6 set jbod J` | `adapter.setJBOD(metadata, "set")` → `runner.Run(["/c0/e251/s6", "set", "jbod"])` |
| `physicaldrives/jbod/disable/fail.json` | `storcli64 /c0/e251/s6 delete jbod J` | `adapter.setJBOD(metadata, "delete")` → `runner.Run(["/c0/e251/s6", "delete", "jbod"])` |
| `logicalvolumes/show/v228.json` | `storcli64 /c0/v228 show all J` | `adapter.showAllVirtualDrive("/c0/v228")` → `runner.Run(["/c0/v228", "show", "all"])` |
| `logicalvolumes/show/all.json` | `storcli64 /c0/v228 show all J` *(same as v228)* | First VD used as "all" reference |
| `logicalvolumes/show/v999_invalid.json` | `storcli64 /c0/v999 show all J` | Error: VD not found |
| `logicalvolumes/create/success.json` | `storcli64 /c0 add vd type=raid0 drives=251:12 rdpolicy=NoRA wrcache=WB iopolicy=Direct J` | `adapter.createLV()` → `runner.Run(["/c0", "add", "vd", ...])` |
| `logicalvolumes/create/fail.json` | `storcli64 /c0 add vd type=raid0 drives=251:1 J` *(drive in use)* | Create error |
| `logicalvolumes/delete/success.json` | `storcli64 /c0/v228 delete J` | `adapter.deleteLV()` → `runner.Run(["/c0/v228", "delete"])` |
| `logicalvolumes/delete/fail_vdNotExist.json` | `storcli64 /c0/v999 delete J` | Delete error: VD doesn't exist |
| `logicalvolumes/delete/fail_invalid.json` | `storcli64 /c0/v299 delete J` | Delete error: invalid VD |
| `logicalvolumes/cacheoptions/success.json` | `storcli64 /c0/v228 set rdcache=RA wrcache=WT iopolicy=Direct J` | `adapter.setLVCacheOptions()` → `runner.Run(["/c0/v228", "set", "rdcache=...", "wrcache=..."])` |
| `logicalvolumes/migrate/fail.json` | `storcli64 /c0/v13 start migrate type=raid0 option=add drives=251:99 J` | `adapter.migrate()` error case |

### storcli2 (`/opt/MegaRAID/storcli2/storcli2`)

| Testdata File | storcli2 Command | Go Code Path |
|---|---|---|
| `controllers/storcli2/all.json` | `storcli2 show all J` | `adapter.showAll()` → `runner.Run(["show", "all"])` |
| `controllers/storcli2/c0.json` | `storcli2 /c0 show all J` | `adapter.showAllController("/c0")` → `runner.Run(["/c0", "show", "all"])` |
| `controllers/storcli2/c0_s12_UGood.json` | `storcli2 /c0 show all J` *(with a drive in UGood state)* | Same as c0.json but pre-VD-creation |
| `controllers/storcli2/c5_invalid.json` | `storcli2 /c5 show all J` | Error: controller not found |
| `physicaldrives/show/storcli2/all.json` | `storcli2 /c0/eall/sall show all J` | All drives detailed info |
| `physicaldrives/show/storcli2/e306s0.json` | `storcli2 /c0/e306/s0 show all J` | `adapter.showAllPhysicalDrive("/c0/e306/s0")` |
| `physicaldrives/show/storcli2/e306s99_invalid.json` | `storcli2 /c0/e306/s99 show all J` | Error: drive not found |
| `physicaldrives/blink/storcli2/start.json` | `storcli2 /c0/e306/s0 start locate J` | `adapter.blink(metadata, "start")` |
| `physicaldrives/blink/storcli2/stop.json` | `storcli2 /c0/e306/s0 stop locate J` | `adapter.blink(metadata, "stop")` |
| `physicaldrives/jbod/enable/storcli2/fail.json` | `storcli2 /c0/e306/s0 set jbod J` | `adapter.setJBOD(metadata, "set")` |
| `physicaldrives/jbod/disable/storcli2/fail.json` | `storcli2 /c0/e306/s0 delete jbod J` | `adapter.setJBOD(metadata, "delete")` |
| `logicalvolumes/show/storcli2/v1.json` | `storcli2 /c0/v1 show all J` | `adapter.showAllVirtualDrive("/c0/v1")` |
| `logicalvolumes/show/storcli2/all.json` | `storcli2 /c0/vall show all J` | All VDs detailed info |
| `logicalvolumes/show/storcli2/v999_invalid.json` | `storcli2 /c0/v999 show all J` | Error: VD not found |
| `logicalvolumes/create/storcli2/success.json` | `storcli2 /c0 add vd type=raid0 drives=306:X rdpolicy=NoRA wrcache=WB iopolicy=Direct J` | `adapter.createLV()` |
| `logicalvolumes/create/storcli2/fail.json` | `storcli2 /c0 add vd type=raid0 drives=306:0 J` *(drive in use)* | Create error |
| `logicalvolumes/delete/storcli2/success.json` | `storcli2 /c0/v{N} delete J` | `adapter.deleteLV()` |
| `logicalvolumes/delete/storcli2/fail_vdNotExist.json` | `storcli2 /c0/v999 delete J` | Delete error: VD doesn't exist |
| `logicalvolumes/delete/storcli2/fail_invalid.json` | `storcli2 /c0/v299 delete J` | Delete error: invalid VD |
| `logicalvolumes/cacheoptions/storcli2/success.json` | `storcli2 /c0/v1 set rdcache=RA wrcache=WT J` | `adapter.setLVCacheOptions()` |
| `logicalvolumes/migrate/storcli2/fail.json` | `storcli2 /c0/v1 start migrate type=raid0 option=add drives=306:99 J` | `adapter.migrate()` error |

## Key Differences between storcli v1 and storcli2

### JSON Output Format Changes

| Feature | storcli v1 | storcli2 |
|---|---|---|
| System Overview controller key | `"Ctl": 0` | `"Ctrl": 0` |
| PD identifier | `"DID": 10` (Device ID) | `"PID": 293` (Persistent ID) |
| PD state | `"State": "Onln"` | `"State": "Conf"`, `"Status": "Online"` |
| VD cache format | `"Cache": "RWTD"` (concatenated) | `"CurrentCache": "NR,WB"`, `"DefaultCache": "NR,WB"` |
| VD LIST fields | `Cache`, `Cac`, `sCC` | `CurrentCache`, `DefaultCache` |
| PD show (single) | `"Drive /c0/e251/s1": [...]` | `"Drives List": [{...}]` |
| PD show detail key | `"Drive /c0/e251/s1 Device attributes"` | Inside `"Drive Detailed Information"` |
| PD SED field | `"SED": "N"` | `"SED_Type": "-"` |
| PD LU/NS info | N/A | `"LU/NS Count": 1`, `"LU/NS Information": [...]` |
| Enclosure list field | `"Port#"`, `"ProdID"` | `"DeviceType"`, `"Partner-EID"`, `"Multipath"` |
| VD show all structure | Flat keys: `/c0/v228`, `PDs for VD 228`, `VD228 Properties` | Nested: `Virtual Drives[0].VD Info`, `Virtual Drives[0].PDs`, `Virtual Drives[0].VD Properties` |
| Controller topology | `"Arr"`, `"DG"` | `"Span"`, `"DG"`, `"PID"` |

### Command Differences

The command syntax is identical between storcli and storcli2. Only the binary name differs:
- storcli: `/opt/MegaRAID/storcli/storcli64`
- storcli2: `/opt/MegaRAID/storcli2/storcli2`

Both use `J` suffix for JSON output.

## How to Regenerate Testdata

### Prerequisites
- Access to a server with the MegaRAID controller and storcli2 installed
- sudo access (storcli2 requires root privileges)

### Steps

1. Copy `collect_storcli2_testdata.sh` to the server
2. Edit the configuration section to match your hardware:
   - `CONTROLLER_INDEX`: your controller ID (typically 0)
   - `ENCLOSURE_IDS`: your enclosure IDs (check with `storcli2 /c0 show all J`)
   - `SLOTS`: slots available in each enclosure
   - `VD_IDS`: virtual drive IDs currently configured
3. Run the script in safe mode first:
   ```bash
   sudo bash collect_storcli2_testdata.sh
   ```
4. For destructive tests (create/delete), configure:
   - `DESTRUCTIVE=true`
   - `CREATE_VD_ENCLOSURE` / `CREATE_VD_SLOT`: an unused drive
5. Copy the output back to this testdata directory

### Capturing UGood State

The `c0_s12_UGood.json` file requires a drive in "unconfigured good" state.
This is the state of a drive that is healthy but not assigned to any virtual drive.

To capture this:
1. Delete a VD to free a drive: `storcli2 /c0/v{N} delete J`
2. Verify the drive shows as unconfigured: `storcli2 /c0 show all J`
3. Run the collection script with `UGOOD_ENCLOSURE` and `UGOOD_SLOT` set
4. Recreate the VD if needed

## Download results

The script will output JSON files in the same structure as this testdata directory. You can copy these files back to your local machine and replace the existing ones in this directory.

```bash
scp -P 10022 -r 'artesca-os@artesca-node:~/storcli2_testdata_output/*' \
  pkg/implementation/raidcontroller/megaraid/testdata/
```

## Known Issues

- **`controllers/storcli2/c0_s12_UGood.json`**: The current file contains the output
  of `storcli2 /c0/eall/sall show all J` (physical drives listing), but it **should**
  contain the output of `storcli2 /c0 show all J` (full controller details, same format
  as `c0.json`) captured when one drive is in UGood/unconfigured state. This needs to be
  recaptured from the server using the collection script with the correct settings.

## storcli2 Commands That Changed Syntax

The following storcli v1 commands do **not work** with storcli2 — the syntax has changed:

### 1. JBOD disable: `delete jbod` is no longer valid

- **storcli v1**: `storcli64 /c0/e251/s6 delete jbod J`
- **storcli2 error**: `syntax error, unexpected TOKEN_JBOD, expecting TOKEN_HOTSPARE_DRIVE`
- **Impact**: The `adapter.setJBOD(metadata, "delete")` code path needs a different
  command for storcli2. The new syntax likely involves a different verb or restructured
  command (possibly related to hotspare/unconfigure operations).
- **TODO**: Investigate storcli2 help/manual for the correct JBOD disable command.

### 2. Online Capacity Expansion (migrate): `start migrate` is no longer valid

- **storcli v1**: `storcli64 /c0/v13 start migrate type=raid0 option=add drives=251:99 J`
- **storcli2 error**: `syntax error, unexpected TOKEN_MIGRATE, expecting TOKEN_ERASE or TOK...`
- **Impact**: The `adapter.migrate()` code path (used by `AddPDsToLV` / `DeletePDsFromLV`)
  needs a different command for storcli2. OCE/migrate may use a completely different
  approach in the storcli2 CLI.
- **TODO**: Investigate storcli2 help/manual for the correct OCE/migration command.

### 3. VD creation: `add vd type=raid0 ... rdpolicy=... wrcache=... iopolicy=...` syntax changed

- **storcli v1**: `storcli64 /c0 add vd type=raid0 drives=251:12 rdpolicy=NoRA wrcache=WB iopolicy=Direct J`
- **storcli2**: `storcli2 /c0 add vd raid0 drives=320:11 wb nora J`
- **Impact**: The `adapter.createLV()` code path needs different argument formatting:
  - `type=raid0` → `raid0` (no `type=` prefix)
  - `rdpolicy=NoRA wrcache=WB iopolicy=Direct` → `wb nora` (shorthand cache flags)
- **Mapping of cache options**:
  - Write Back: `wrcache=WB` → `wb`
  - Write Through: `wrcache=WT` → `wt`
  - No Read Ahead: `rdpolicy=NoRA` → `nora`
  - Read Ahead: `rdpolicy=RA` → `ra`

### 4. VD cache options: `set rdcache=RA wrcache=WT` combined syntax no longer valid

- **storcli v1**: `storcli64 /c0/v228 set rdcache=RA wrcache=WT iopolicy=Direct J`
- **storcli2 error**: `syntax error, unexpected TOKEN_WRITE_CACHE, expecting $end`
- **Impact**: The `adapter.setLVCacheOptions()` code path needs to issue **separate commands**
  for each cache option in storcli2:
  - `storcli2 /c0/v24 set wrcache=WT J` (set write cache separately)
  - `storcli2 /c0/v24 set rdcache=RA J` (set read cache separately)
- **`iopolicy` is removed in storcli2** — it does not appear in `storcli2 /cx/vx set help`.
  The adapter code should skip setting iopolicy when using storcli2.

> **Note**: Both commands produce plain-text errors (not JSON) when using invalid syntax,
> which means the Go `Runner.Run()` method will fail to parse the output. The adapter
> implementation for storcli2 will need to handle these operations differently.

## Notes

- All JSON files should be valid JSON (no trailing characters, proper encoding)
- The `J` flag at the end of storcli/storcli2 commands produces JSON output
- Some error cases return non-zero exit codes but still produce valid JSON
- Files in `logicalvolumes/show/all_NOv228.json` is a variant without VD 228
  (used to test the createLV flow which needs to detect the newly created VD)
