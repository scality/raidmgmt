# storcli2 / perccli2 testdata tooling

This directory holds the helper script used to capture JSON fixtures from the
second-generation MegaRAID CLI (`storcli2` / `perccli2`). The captured fixtures
back the unit tests of the decomposed storcli2 adapter components.

## collect_storcli2_testdata.sh

`collect_storcli2_testdata.sh` runs a series of `storcli2` commands on a host
that has a MegaRAID controller and stores their JSON output as fixture files.

It is adapted from the original capture script (PR #46): the command logic is
unchanged, but the output is written directly into the decomposed
**per-component** layout this repository uses, so the result can be copied back
verbatim. The output tree mirrors `pkg/implementation/<component>/testdata/storcli2/`.

Usage:

1. Copy the script to a server with `storcli2` installed.
2. Edit the configuration block at the top (binary path, controller index,
   enclosure IDs, slots, virtual-drive IDs, invalid IDs).
3. Run it as root: `sudo bash collect_storcli2_testdata.sh`.
   - SAFE mode (default) collects read-only `show` outputs only.
   - Set `DESTRUCTIVE=true` to also collect create / delete / migrate / cache /
     JBOD outputs (these modify the array or a drive LED — use a scratch host).
4. Copy the generated tree back into the repository:
   `cp -r ./storcli2_testdata_output/* pkg/implementation/`

## Fixture layout

The script writes each fixture under its owning component package's
`testdata/storcli2/` directory:

| Package | Path | storcli2 command | Mode |
|---|---|---|---|
| `controllergetter` | `testdata/storcli2/all.json` | `show all` | safe |
| `controllergetter` | `testdata/storcli2/c0.json` | `/c0 show all` | safe |
| `controllergetter` | `testdata/storcli2/c5_invalid.json` | `/c5 show all` (controller not found) | safe |
| `controllergetter` | `testdata/storcli2/c0_s12_UGood.json` | `/c0 show all` (a drive in UGood state) | destructive |
| `physicaldrivegetter` | `testdata/storcli2/show/all.json` | `/c0/eall/sall show all` | safe |
| `physicaldrivegetter` | `testdata/storcli2/show/e3{06,20}sN.json` | `/c0/e3XX/sN show all` | safe |
| `physicaldrivegetter` | `testdata/storcli2/show/e306s99_invalid.json` | drive not found | safe |
| `physicaldrivegetter` | `testdata/storcli2/show/e320s11_UGood.json` | drive in unconfigured-good state | destructive |
| `physicaldrivegetter` | `testdata/storcli2/jbod/{enable,disable}/fail.json` | `set jbod` / `delete jbod` | destructive |
| `logicalvolumegetter` | `testdata/storcli2/show/all.json` | `/c0/vall show all` | safe |
| `logicalvolumegetter` | `testdata/storcli2/show/vN.json` | `/c0/vN show all` | safe |
| `logicalvolumegetter` | `testdata/storcli2/show/v999_invalid.json` | VD not found | safe |
| `logicalvolumemanager` | `testdata/storcli2/create/{success,fail}.json` | `add vd ...` | destructive |
| `logicalvolumemanager` | `testdata/storcli2/delete/{success,fail_invalid,fail_vdNotExist}.json` | `delete` | destructive / safe |
| `logicalvolumemanager` | `testdata/storcli2/migrate/fail.json` | `start migrate ...` | destructive |
| `logicalvolumemanager` | `testdata/storcli2/cacheoptions/success*.json` | `set rdcache/wrcache` | destructive |
| `blinker` | `testdata/storcli2/{start,stop}.json` | `start locate` / `stop locate` | destructive |

The envelope / decoder unit tests in `pkg/implementation/storcli2` keep their own
curated copies under that package's `testdata/` directory.

## Known issues with the captured data

The following fixtures were captured as plain-text syntax errors rather than
JSON, because `storcli2` changed the CLI grammar for these commands relative to
storcli v1 (the script still uses the v1 syntax):

- `logicalvolumemanager/testdata/storcli2/cacheoptions/success.json`
  (`unexpected TOKEN_WRITE_CACHE`)
- `logicalvolumemanager/testdata/storcli2/migrate/fail.json`
  (`unexpected TOKEN_MIGRATE`)
- `physicaldrivegetter/testdata/storcli2/jbod/disable/fail.json`
  (`unexpected TOKEN_JBOD`)

These must be regenerated with the correct storcli2 syntax when the
corresponding manager / JBOD-setter components are implemented. Until then they
are not valid runner inputs.
