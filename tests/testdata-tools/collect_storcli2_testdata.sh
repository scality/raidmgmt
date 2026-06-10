#!/bin/bash
# =============================================================================
# Script: collect_storcli2_testdata.sh
#
# Purpose: Collect JSON outputs from storcli2 to generate testdata files for
#          the raidmgmt project's storcli2 support.
#
# Usage:
#   1. Copy this script to the server with storcli2 installed
#   2. Configure the variables below to match your environment
#   3. Run: sudo bash collect_storcli2_testdata.sh
#   4. Copy the output back into the repository's pkg/implementation/ (the
#      output mirrors each component package's testdata/storcli2 layout)
#
# Prerequisites:
#   - storcli2 installed at STORCLI2_PATH
#   - sudo access (storcli2 requires root)
#   - At least one RAID controller present
#   - For "UGood" variants: one drive must be unconfigured (not in a VD)
#   - For create/delete tests: ability to create/delete a VD (destructive!)
#
# The script has two modes:
#   - SAFE mode (default): Collects read-only outputs only
#   - DESTRUCTIVE mode: Also collects create/delete/migrate outputs
#     Set DESTRUCTIVE=true to enable
#
# =============================================================================

set -euo pipefail

# =============================================================================
# CONFIGURATION - Edit these variables to match your environment
# =============================================================================

# Path to the storcli2 binary
STORCLI2_PATH="/opt/MegaRAID/storcli2/storcli2"

# Controller index (typically 0)
CONTROLLER_INDEX=0

# Enclosure IDs (space-separated list)
# From your setup: enclosures 306 and 320
ENCLOSURE_IDS="306 320"

# Slots per enclosure (space-separated list)
# From your setup: slots 0-11 for each enclosure
SLOTS="0 1 2 3 4 5 6 7 8 9 10 11"

# Virtual Drive IDs to capture (space-separated)
# From your setup: VDs 1-24 (NOTE: if a VD was deleted and not recreated,
# storcli2 will still return exit 0 with "Invalid VD number" in JSON).
# Consider removing deleted VD IDs from this list.
VD_IDS="1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23"

# Invalid controller index (for error test cases)
INVALID_CONTROLLER_INDEX=5

# Invalid VD ID (for error test cases)
INVALID_VD_ID=999

# Invalid slot (for error test cases)
INVALID_SLOT=99

# First enclosure (used for invalid slot test and blink tests)
FIRST_ENCLOSURE=306

# First slot (used for blink and JBOD tests)
FIRST_SLOT=0

# Enable destructive operations (create/delete VDs)
# WARNING: This will temporarily delete and recreate a virtual drive!
# In destructive mode the script will:
#   1. Delete SACRIFICE_VD to free its drive (UGood state)
#   2. Capture controller & drive outputs in UGood state
#   3. Recreate the VD and capture create/cache/delete outputs
DESTRUCTIVE=false

# VD to sacrifice for UGood capture (will be deleted then recreated)
# Pick a non-critical VD. The script will auto-detect its drive location.
# Set to "auto" to use the last VD from the controller's VD LIST.
# Or set to a specific VD ID (e.g. "24").
SACRIFICE_VD_ID="auto"

# Output directory
OUTPUT_DIR="./storcli2_testdata_output"

# =============================================================================
# END CONFIGURATION
# =============================================================================

# Shorthand
CLI="${STORCLI2_PATH}"
C="/c${CONTROLLER_INDEX}"

echo "============================================"
echo " storcli2 Testdata Collection Script"
echo "============================================"
echo ""
echo "Configuration:"
echo "  CLI:         ${CLI}"
echo "  Controller:  ${CONTROLLER_INDEX}"
echo "  Enclosures:  ${ENCLOSURE_IDS}"
echo "  Output:      ${OUTPUT_DIR}"
echo "  Destructive: ${DESTRUCTIVE}"
echo ""

# Verify storcli2 exists
if [ ! -x "${CLI}" ]; then
    echo "ERROR: storcli2 not found at ${CLI}"
    exit 1
fi

# Create output directories
mkdir -p "${OUTPUT_DIR}/controllergetter/testdata/storcli2"
mkdir -p "${OUTPUT_DIR}/physicaldrivegetter/testdata/storcli2/show"
mkdir -p "${OUTPUT_DIR}/blinker/testdata/storcli2"
mkdir -p "${OUTPUT_DIR}/physicaldrivegetter/testdata/storcli2/jbod/enable"
mkdir -p "${OUTPUT_DIR}/physicaldrivegetter/testdata/storcli2/jbod/disable"
mkdir -p "${OUTPUT_DIR}/logicalvolumegetter/testdata/storcli2/show"
mkdir -p "${OUTPUT_DIR}/logicalvolumemanager/testdata/storcli2/create"
mkdir -p "${OUTPUT_DIR}/logicalvolumemanager/testdata/storcli2/delete"
mkdir -p "${OUTPUT_DIR}/logicalvolumemanager/testdata/storcli2/cacheoptions"
mkdir -p "${OUTPUT_DIR}/logicalvolumemanager/testdata/storcli2/migrate"

# Helper function to run a command and save output
run_and_save() {
    local description="$1"
    local output_file="$2"
    shift 2
    local cmd_args=("$@")

    echo "  [COLLECT] ${description}"
    echo "            Command: ${CLI} ${cmd_args[*]} J"
    echo "            Output:  ${output_file}"

    # Run command and save. Capture both stdout and stderr since storcli2
    # may write JSON errors to stderr on some failure cases.
    # Allow non-zero exit codes (expected for error test cases).
    if "${CLI}" "${cmd_args[@]}" J > "${output_file}" 2>&1; then
        echo "            Status:  OK"
    else
        echo "            Status:  Command returned non-zero (may be expected for error cases)"
    fi

    # Verify the output is valid JSON; if not, try to salvage or flag it
    if [ ! -s "${output_file}" ]; then
        echo "            WARNING: Output file is empty!"
        rm -f "${output_file}"
    elif ! python3 -m json.tool "${output_file}" > /dev/null 2>&1; then
        # storcli2 may mix text preamble with JSON — try extracting JSON portion
        # Look for first '{' to last '}'
        if python3 -c "
import sys, json, re
with open('${output_file}', 'r') as f:
    content = f.read()
start = content.find('{')
end = content.rfind('}')
if start >= 0 and end > start:
    obj = json.loads(content[start:end+1])
    with open('${output_file}', 'w') as f:
        json.dump(obj, f, indent=2)
else:
    sys.exit(1)
" 2>/dev/null; then
            echo "            WARNING: Extracted JSON from mixed output"
        else
            echo "            WARNING: Output is not valid JSON!"
            echo "            Content: $(head -c 200 "${output_file}")"
        fi
    fi
    echo ""
}

# =============================================================================
# 1. CONTROLLERS
# =============================================================================
echo "--- Controllers ---"
echo ""

# controllergetter/testdata/storcli2/all.json
# Command: storcli2 show all J
# Used by: adapter.showAll() → runner.Run(["show", "all"])
run_and_save \
    "All controllers overview" \
    "${OUTPUT_DIR}/controllergetter/testdata/storcli2/all.json" \
    show all

# controllergetter/testdata/storcli2/c0.json
# Command: storcli2 /c0 show all J
# Used by: adapter.showAllController("/c0") → runner.Run(["/c0", "show", "all"])
run_and_save \
    "Controller ${CONTROLLER_INDEX} full details" \
    "${OUTPUT_DIR}/controllergetter/testdata/storcli2/c0.json" \
    "${C}" show all

# controllergetter/testdata/storcli2/c0_aso.json
# Command: storcli2 /c0 show aso J
# Used by: StorCLI2.jbodSupported("/c0") → runner.Run(["/c0", "show", "aso"])
# JBOD capability = a non-expired "JBOD" Advanced Software Option (license).
run_and_save \
    "Controller ${CONTROLLER_INDEX} advanced software options" \
    "${OUTPUT_DIR}/controllergetter/testdata/storcli2/c0_aso.json" \
    "${C}" show aso

# controllergetter/testdata/storcli2/c0_autoconfig.json
# Command: storcli2 /c0 show autoconfig J
# Used by: StorCLI2.jbodEnabled("/c0") → runner.Run(["/c0", "show", "autoconfig"])
# JBOD active = Primary Auto-configure behavior is JBOD or SecureJBOD.
run_and_save \
    "Controller ${CONTROLLER_INDEX} auto-configure behavior" \
    "${OUTPUT_DIR}/controllergetter/testdata/storcli2/c0_autoconfig.json" \
    "${C}" show autoconfig

# controllergetter/testdata/storcli2/c5_invalid.json
# Command: storcli2 /c5 show all J
# Used by: error case when controller doesn't exist
run_and_save \
    "Invalid controller (expected failure)" \
    "${OUTPUT_DIR}/controllergetter/testdata/storcli2/c5_invalid.json" \
    "/c${INVALID_CONTROLLER_INDEX}" show all

# controllergetter/testdata/storcli2/c0_s12_UGood.json
# Command: storcli2 /c0 show all J (with one drive in UGood state)
# Used by: createLV flow - controller must have an unconfigured drive
# NOTE: Captured automatically in DESTRUCTIVE mode (delete VD → capture → recreate)
echo "  [SKIP] c0_s12_UGood.json will be captured in DESTRUCTIVE mode"
echo ""

# =============================================================================
# 2. PHYSICAL DRIVES
# =============================================================================
echo "--- Physical Drives ---"
echo ""

# physicaldrivegetter/testdata/storcli2/show/all.json
# Command: storcli2 /c0/eall/sall show all J
# Used by: display of all drives with full details
run_and_save \
    "All physical drives (all enclosures, all slots)" \
    "${OUTPUT_DIR}/physicaldrivegetter/testdata/storcli2/show/all.json" \
    "${C}/eall/sall" show all

# Individual physical drives
# Command: storcli2 /c0/e{EID}/s{SLOT} show all J
# Used by: adapter.showAllPhysicalDrive("/c0/e306/s0") → runner.Run(["/c0/e306/s0", "show", "all"])
for eid in ${ENCLOSURE_IDS}; do
    for slot in ${SLOTS}; do
        run_and_save \
            "Physical drive e${eid}/s${slot}" \
            "${OUTPUT_DIR}/physicaldrivegetter/testdata/storcli2/show/e${eid}s${slot}.json" \
            "${C}/e${eid}/s${slot}" show all
    done
done

# Invalid physical drive (error case)
# Command: storcli2 /c0/e306/s99 show all J
# Used by: error case when drive doesn't exist
run_and_save \
    "Invalid physical drive (expected failure)" \
    "${OUTPUT_DIR}/physicaldrivegetter/testdata/storcli2/show/e${FIRST_ENCLOSURE}s${INVALID_SLOT}_invalid.json" \
    "${C}/e${FIRST_ENCLOSURE}/s${INVALID_SLOT}" show all

# Blink start
# Command: storcli2 /c0/e306/s0 start locate J
# Used by: adapter.blink(metadata, "start") → runner.Run(["/c0/e306/s0", "start", "locate"])
run_and_save \
    "Start blink on e${FIRST_ENCLOSURE}/s${FIRST_SLOT}" \
    "${OUTPUT_DIR}/blinker/testdata/storcli2/start.json" \
    "${C}/e${FIRST_ENCLOSURE}/s${FIRST_SLOT}" start locate

# Blink stop
# Command: storcli2 /c0/e306/s0 stop locate J
# Used by: adapter.blink(metadata, "stop") → runner.Run(["/c0/e306/s0", "stop", "locate"])
run_and_save \
    "Stop blink on e${FIRST_ENCLOSURE}/s${FIRST_SLOT}" \
    "${OUTPUT_DIR}/blinker/testdata/storcli2/stop.json" \
    "${C}/e${FIRST_ENCLOSURE}/s${FIRST_SLOT}" stop locate

# JBOD enable (expected failure when drive is in a VD)
# Command: storcli2 /c0/e306/s0 set jbod J
# Used by: adapter.setJBOD(metadata, "set") → runner.Run(["/c0/e306/s0", "set", "jbod"])
run_and_save \
    "Enable JBOD on e${FIRST_ENCLOSURE}/s${FIRST_SLOT} (expected failure - drive in VD)" \
    "${OUTPUT_DIR}/physicaldrivegetter/testdata/storcli2/jbod/enable/fail.json" \
    "${C}/e${FIRST_ENCLOSURE}/s${FIRST_SLOT}" set jbod

# JBOD disable (expected failure when drive is in a VD)
# Command: storcli2 /c0/e306/s0 delete jbod J
# Used by: adapter.setJBOD(metadata, "delete") → runner.Run(["/c0/e306/s0", "delete", "jbod"])
run_and_save \
    "Disable JBOD on e${FIRST_ENCLOSURE}/s${FIRST_SLOT} (expected failure - drive in VD)" \
    "${OUTPUT_DIR}/physicaldrivegetter/testdata/storcli2/jbod/disable/fail.json" \
    "${C}/e${FIRST_ENCLOSURE}/s${FIRST_SLOT}" delete jbod

# =============================================================================
# 3. LOGICAL VOLUMES
# =============================================================================
echo "--- Logical Volumes ---"
echo ""

# Individual virtual drive details
# Command: storcli2 /c0/v{N} show all J
# Used by: adapter.showAllVirtualDrive("/c0/v1") → runner.Run(["/c0/v1", "show", "all"])
for vd_id in ${VD_IDS}; do
    run_and_save \
        "Virtual drive v${vd_id}" \
        "${OUTPUT_DIR}/logicalvolumegetter/testdata/storcli2/show/v${vd_id}.json" \
        "${C}/v${vd_id}" show all
done

# "all" variant - same content as first VD but used as the reference "all" file
# In storcli v1, this was a multi-VD output. For storcli2, capture all VDs:
# Command: storcli2 /c0/vall show all J
# Used by: listing all logical volumes with their details
run_and_save \
    "All virtual drives" \
    "${OUTPUT_DIR}/logicalvolumegetter/testdata/storcli2/show/all.json" \
    "${C}/vall" show all

# Invalid VD (error case)
# Command: storcli2 /c0/v999 show all J
# Used by: error case when VD doesn't exist
run_and_save \
    "Invalid virtual drive (expected failure)" \
    "${OUTPUT_DIR}/logicalvolumegetter/testdata/storcli2/show/v${INVALID_VD_ID}_invalid.json" \
    "${C}/v${INVALID_VD_ID}" show all

# Delete VD - failure case (VD doesn't exist)
# Command: storcli2 /c0/v999 delete J
# Used by: adapter.deleteLV() error case
run_and_save \
    "Delete non-existent VD (expected failure)" \
    "${OUTPUT_DIR}/logicalvolumemanager/testdata/storcli2/delete/fail_vdNotExist.json" \
    "${C}/v${INVALID_VD_ID}" delete

# Migrate - failure case (operation not supported/possible)
# Command: storcli2 /c0/v1 start migrate type=raid0 option=add drives=306:99 J
# Used by: adapter.migrate() error case
run_and_save \
    "Migrate VD (expected failure)" \
    "${OUTPUT_DIR}/logicalvolumemanager/testdata/storcli2/migrate/fail.json" \
    "${C}/v1" start migrate type=raid0 option=add "drives=${FIRST_ENCLOSURE}:${INVALID_SLOT}"

# =============================================================================
# 4. DESTRUCTIVE OPERATIONS (disabled by default)
#    Flow: delete VD → capture UGood state → recreate VD → capture create/cache/delete
# =============================================================================
if [ "${DESTRUCTIVE}" = "true" ]; then
    echo "--- Destructive Operations ---"
    echo ""

    # --- Step 0: Auto-detect sacrifice VD if needed ---
    if [ "${SACRIFICE_VD_ID}" = "auto" ]; then
        echo "  [STEP 0] Auto-detecting sacrifice VD (last VD in controller)..."
        SACRIFICE_VD_ID=$("${CLI}" "${C}" show all J 2>&1 | python3 -c "
import sys, json

content = sys.stdin.read()
start = content.find('{')
end = content.rfind('}')
if start < 0 or end <= start:
    sys.exit(1)
data = json.loads(content[start:end+1])
rd = data['Controllers'][0]['Response Data']

# Get last VD from VD LIST
vd_list = rd.get('VD LIST', [])
if isinstance(vd_list, list) and vd_list:
    last = vd_list[-1]
    if isinstance(last, dict):
        dgvd = last.get('DG/VD', '')
        if '/' in str(dgvd):
            print(str(dgvd).split('/')[1])
            sys.exit(0)
sys.exit(1)
" 2>/dev/null || echo "")

        if [ -z "${SACRIFICE_VD_ID}" ]; then
            echo "  [ERROR] Could not auto-detect sacrifice VD from controller."
            echo "          Set SACRIFICE_VD_ID manually in the script."
            SACRIFICE_VD_ID=""
        else
            echo "           Auto-detected: v${SACRIFICE_VD_ID}"
            echo ""
        fi
    fi

    if [ -z "${SACRIFICE_VD_ID}" ]; then
        echo "  [ERROR] No valid SACRIFICE_VD_ID. Skipping destructive operations."
        echo ""
    else

    echo "  [INFO] Sacrifice VD: v${SACRIFICE_VD_ID}"
    echo ""

    # --- Step 1: Identify the drive behind the sacrifice VD ---
    echo "  [STEP 1] Detecting drive behind v${SACRIFICE_VD_ID}..."
    VD_INFO_JSON=$("${CLI}" "${C}/v${SACRIFICE_VD_ID}" show all J 2>&1 || true)

    # Verify the VD actually exists (storcli2 returns exit 0 even for invalid VDs)
    VD_STATUS=$(echo "${VD_INFO_JSON}" | python3 -c "
import sys, json
content = sys.stdin.read()
start = content.find('{')
end = content.rfind('}')
if start < 0 or end <= start:
    sys.exit(1)
data = json.loads(content[start:end+1])
status = data['Controllers'][0]['Command Status']['Status']
print(status)
" 2>/dev/null || echo "Unknown")

    if [ "${VD_STATUS}" != "Success" ]; then
        echo "  [ERROR] VD v${SACRIFICE_VD_ID} does not exist (Status: ${VD_STATUS})."
        echo "          The VD may have been deleted in a previous run."
        echo "          Either recreate it or set SACRIFICE_VD_ID to a valid VD."
        echo ""
        echo "          Available VDs can be found with: storcli2 /c0/vall show J"
        echo ""
    else

    # Parse enclosure:slot of the first PD in this VD
    # storcli2 format: Response Data > Virtual Drives[0] > PDs[0] > EID:Slt (or EID:Slot)
    SACRIFICE_DRIVE=$(echo "${VD_INFO_JSON}" | python3 -c "
import sys, json

content = sys.stdin.read()
start = content.find('{')
end = content.rfind('}')
if start < 0 or end <= start:
    sys.exit(1)
data = json.loads(content[start:end+1])
rd = data['Controllers'][0]['Response Data']

def get_eid_slt(pd):
    '''Get EID:Slot value from PD dict, trying both field name variants.'''
    return pd.get('EID:Slt') or pd.get('EID:Slot') or ''

# storcli2 format: nested under 'Virtual Drives' array
vds = rd.get('Virtual Drives', [])
if isinstance(vds, list) and vds:
    for vd in vds:
        if not isinstance(vd, dict):
            continue
        pds = vd.get('PDs', [])
        if isinstance(pds, list) and pds:
            val = get_eid_slt(pds[0])
            if val and val != '-':
                print(val)
                sys.exit(0)

# Fallback: storcli v1 format - 'PDs for VD N' key
for key in rd:
    if key.startswith('PDs for VD'):
        pds = rd[key]
        if isinstance(pds, list) and pds:
            val = get_eid_slt(pds[0])
            if val and val != '-':
                print(val)
                sys.exit(0)

# Fallback: search all nested dicts/lists for any PD-like object
def search_pds(obj):
    if isinstance(obj, dict):
        val = get_eid_slt(obj)
        if val and val != '-' and ':' in val:
            return val
        for v in obj.values():
            result = search_pds(v)
            if result:
                return result
    elif isinstance(obj, list):
        for item in obj:
            result = search_pds(item)
            if result:
                return result
    return None

result = search_pds(rd)
if result:
    print(result)
    sys.exit(0)

sys.exit(1)
" 2>/dev/null || echo "")

    if [ -z "${SACRIFICE_DRIVE}" ]; then
        # Fallback: Try to find the drive from controller TOPOLOGY
        echo "  [STEP 1b] VD show all parsing failed, trying controller TOPOLOGY..."
        SACRIFICE_DRIVE=$("${CLI}" "${C}" show all J 2>&1 | python3 -c "
import sys, json

content = sys.stdin.read()
start = content.find('{')
end = content.rfind('}')
if start < 0 or end <= start:
    sys.exit(1)
data = json.loads(content[start:end+1])
rd = data['Controllers'][0]['Response Data']

# Find the DG for the target VD from VD LIST
target_vd = ${SACRIFICE_VD_ID}
target_dg = None
vd_list = rd.get('VD LIST', [])
for vd in vd_list:
    dgvd = vd.get('DG/VD', '')
    if '/' in str(dgvd):
        parts = str(dgvd).split('/')
        if parts[1] == str(target_vd):
            target_dg = int(parts[0])
            break

if target_dg is None:
    sys.exit(1)

# Find the drive for this DG from TOPOLOGY
topology = rd.get('TOPOLOGY', [])
for entry in topology:
    if not isinstance(entry, dict):
        continue
    if entry.get('DG') == target_dg and entry.get('Type') == 'DRIVE':
        eid_slot = entry.get('EID:Slot', '') or entry.get('EID:Slt', '')
        if eid_slot and eid_slot != '-' and ':' in eid_slot:
            print(eid_slot)
            sys.exit(0)

sys.exit(1)
" 2>/dev/null || echo "")
    fi

    if [ -z "${SACRIFICE_DRIVE}" ]; then
        echo "  [ERROR] Could not detect drive for v${SACRIFICE_VD_ID}."
        echo "          Make sure SACRIFICE_VD_ID is valid and the VD exists."
        echo ""
        echo "          Debug: dumping first 500 chars of VD JSON:"
        echo "${VD_INFO_JSON}" | head -c 500
        echo ""
        echo ""
    else
        # Parse enclosure and slot from "EID:Slot" format (e.g. "320:11")
        SACRIFICE_EID=$(echo "${SACRIFICE_DRIVE}" | cut -d: -f1)
        SACRIFICE_SLOT=$(echo "${SACRIFICE_DRIVE}" | cut -d: -f2)
        echo "           Drive: e${SACRIFICE_EID}/s${SACRIFICE_SLOT}"
        echo ""

        # --- Step 2: Delete the sacrifice VD to free the drive ---
        echo "  [STEP 2] Deleting v${SACRIFICE_VD_ID} to free drive..."
        run_and_save \
            "Delete VD v${SACRIFICE_VD_ID} (success case)" \
            "${OUTPUT_DIR}/logicalvolumemanager/testdata/storcli2/delete/success.json" \
            "${C}/v${SACRIFICE_VD_ID}" delete

        # Give the controller a moment to settle
        sleep 2

        # --- Step 3: Capture UGood state ---
        echo "  [STEP 3] Capturing UGood state..."

        # controllergetter/testdata/storcli2/c0_s12_UGood.json
        # Command: storcli2 /c0 show all J (with drive in UGood/unconfigured state)
        # Used by: createLV flow - controller must show an unconfigured drive
        run_and_save \
            "Controller with UGood drive (c0 show all)" \
            "${OUTPUT_DIR}/controllergetter/testdata/storcli2/c0_s12_UGood.json" \
            "${C}" show all

        # physicaldrivegetter/testdata/storcli2/show/e{EID}s{SLOT}_UGood.json
        # Command: storcli2 /c0/e{EID}/s{SLOT} show all J (drive in UGood state)
        # Used by: physical drive detail when drive is unconfigured
        run_and_save \
            "Physical drive e${SACRIFICE_EID}/s${SACRIFICE_SLOT} in UGood state" \
            "${OUTPUT_DIR}/physicaldrivegetter/testdata/storcli2/show/e${SACRIFICE_EID}s${SACRIFICE_SLOT}_UGood.json" \
            "${C}/e${SACRIFICE_EID}/s${SACRIFICE_SLOT}" show all

        # --- Step 4: Recreate the VD (captures create success) ---
        echo "  [STEP 4] Recreating VD on e${SACRIFICE_EID}/s${SACRIFICE_SLOT}..."

        # logicalvolumemanager/testdata/storcli2/create/success.json
        # Command: storcli2 /c0 add vd raid0 drives=EID:SLOT wb nora J
        # Used by: adapter.createLV() → runner.Run(["/c0", "add", "vd", ...])
        # NOTE: storcli2 uses different syntax than v1!
        #   v1:  /c0 add vd type=raid0 drives=EID:SLOT rdpolicy=NoRA wrcache=WB iopolicy=Direct
        #   v2:  /c0 add vd raid0 drives=EID:SLOT wb nora
        run_and_save \
            "Create VD (RAID0 on e${SACRIFICE_EID}/s${SACRIFICE_SLOT})" \
            "${OUTPUT_DIR}/logicalvolumemanager/testdata/storcli2/create/success.json" \
            "${C}" add vd raid0 "drives=${SACRIFICE_EID}:${SACRIFICE_SLOT}" wb nora

        sleep 2

        # --- Step 5: Detect the newly created VD ID ---
        echo "  [STEP 5] Detecting newly created VD..."
        NEWEST_VD=$("${CLI}" "${C}/vall" show J 2>&1 | python3 -c "
import sys, json
content = sys.stdin.read()
start = content.find('{')
end = content.rfind('}')
if start < 0 or end <= start:
    sys.exit(1)
data = json.loads(content[start:end+1])
rd = data['Controllers'][0]['Response Data']

# storcli2 format: 'Virtual Drives' array with 'VD Info' sub-objects
vds = rd.get('Virtual Drives', [])
if isinstance(vds, list) and vds:
    last_vd = vds[-1]
    if isinstance(last_vd, dict):
        # May be nested under 'VD Info' or directly have 'DG/VD'
        vd_info = last_vd.get('VD Info', last_vd)
        if isinstance(vd_info, dict):
            dgvd = vd_info.get('DG/VD', '')
            if '/' in str(dgvd):
                print(str(dgvd).split('/')[1])
                sys.exit(0)
        # If VD Info is a list, get the first element
        elif isinstance(vd_info, list) and vd_info:
            dgvd = vd_info[0].get('DG/VD', '')
            if '/' in str(dgvd):
                print(str(dgvd).split('/')[1])
                sys.exit(0)

# Fallback: storcli v1/v2 flat 'VD LIST' format
vd_list = rd.get('VD LIST', [])
if isinstance(vd_list, list) and vd_list:
    last = vd_list[-1]
    if isinstance(last, dict):
        dgvd = last.get('DG/VD', '')
        if '/' in str(dgvd):
            print(str(dgvd).split('/')[1])
            sys.exit(0)

sys.exit(1)
" 2>/dev/null || echo "")

        if [ -n "${NEWEST_VD}" ]; then
            echo "           New VD ID: v${NEWEST_VD}"
            echo ""

            # --- Step 6: Cache options success ---
            # logicalvolumemanager/testdata/storcli2/cacheoptions/success.json
            # storcli2 uses separate commands for each cache option
            # (v1 combined syntax "set rdcache=RA wrcache=WT" does not work)
            # Try: storcli2 /c0/v{N} set wrcache=WT J
            #      storcli2 /c0/v{N} set rdcache=RA J
            # Used by: adapter.setLVCacheOptions()
            run_and_save \
                "Set write cache on v${NEWEST_VD}" \
                "${OUTPUT_DIR}/logicalvolumemanager/testdata/storcli2/cacheoptions/success_wrcache.json" \
                "${C}/v${NEWEST_VD}" set wrcache=WT

            run_and_save \
                "Set read cache on v${NEWEST_VD}" \
                "${OUTPUT_DIR}/logicalvolumemanager/testdata/storcli2/cacheoptions/success_rdcache.json" \
                "${C}/v${NEWEST_VD}" set rdcache=RA

            # Also try combined (may fail - for documentation purposes)
            run_and_save \
                "Set cache options combined on v${NEWEST_VD} (may fail in storcli2)" \
                "${OUTPUT_DIR}/logicalvolumemanager/testdata/storcli2/cacheoptions/success.json" \
                "${C}/v${NEWEST_VD}" set rdcache=RA wrcache=WT
        else
            echo "  [ERROR] Could not determine newly created VD ID"
            echo ""
        fi

        # --- Step 7: Create failure case (drive already in use) ---
        # logicalvolumemanager/testdata/storcli2/create/fail.json
        # Command: storcli2 /c0 add vd raid0 drives=EID:SLOT J (drive now in use)
        # Used by: adapter.createLV() error case
        run_and_save \
            "Create VD on in-use drive (expected failure)" \
            "${OUTPUT_DIR}/logicalvolumemanager/testdata/storcli2/create/fail.json" \
            "${C}" add vd raid0 "drives=${FIRST_ENCLOSURE}:${FIRST_SLOT}"

        # --- Step 8: Delete failure cases ---
        # logicalvolumemanager/testdata/storcli2/delete/fail_invalid.json
        # Command: storcli2 /c0/v299 delete J
        run_and_save \
            "Delete invalid VD (expected failure)" \
            "${OUTPUT_DIR}/logicalvolumemanager/testdata/storcli2/delete/fail_invalid.json" \
            "${C}/v299" delete

        echo "  [DONE] Destructive operations complete."
        echo "         VD was recreated on e${SACRIFICE_EID}/s${SACRIFICE_SLOT}."
        echo "         New VD ID: v${NEWEST_VD:-unknown}"
        echo ""
    fi
    fi # end VD_STATUS check
    fi # end SACRIFICE_VD_ID empty check
else
    echo "--- Destructive Operations SKIPPED (set DESTRUCTIVE=true to enable) ---"
    echo ""
    echo "  The following files need destructive operations to capture:"
    echo "    - controllergetter/testdata/storcli2/c0_s12_UGood.json (controller with UGood drive)"
    echo "    - physicaldrivegetter/testdata/storcli2/show/e{EID}s{SLOT}_UGood.json"
    echo "    - logicalvolumemanager/testdata/storcli2/create/success.json"
    echo "    - logicalvolumemanager/testdata/storcli2/create/fail.json"
    echo "    - logicalvolumemanager/testdata/storcli2/delete/success.json"
    echo "    - logicalvolumemanager/testdata/storcli2/delete/fail_invalid.json"
    echo "    - logicalvolumemanager/testdata/storcli2/cacheoptions/success.json"
    echo ""
fi

# =============================================================================
# DONE
# =============================================================================
echo "============================================"
echo " Collection Complete!"
echo "============================================"
echo ""
echo "Output directory: ${OUTPUT_DIR}"
echo ""
echo "To use these files in the project:"
echo "  1. Copy the contents of ${OUTPUT_DIR}/ into pkg/implementation/."
echo "     The output mirrors each component package's testdata/storcli2 layout:"
echo "       cp -r ${OUTPUT_DIR}/* pkg/implementation/"
echo ""
echo "  2. Verify JSON files are valid:"
echo "     find ${OUTPUT_DIR} -name '*.json' -exec python3 -m json.tool {} > /dev/null \;"
echo ""
echo "Files collected:"
find "${OUTPUT_DIR}" -name "*.json" -type f | sort | while read -r f; do
    echo "  ${f}"
done
