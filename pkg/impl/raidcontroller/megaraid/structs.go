package megaraid

import (
	"bytes"
	"encoding/json"
	"strings"
)

type (
	CmdOutput struct {
		Controllers []Controllers `json:"Controllers"`
	}

	Controllers struct {
		CommandStatus CommandStatus   `json:"Command Status"`
		ResponseData  json.RawMessage `json:"Response Data"`
	}

	CommandStatus struct {
		CLIVersion      string           `json:"CLI Version"`
		OperatingSystem string           `json:"Operating system"`
		StatusCode      int              `json:"Status Code"`
		Status          string           `json:"Status"`
		Description     string           `json:"Description"`
		Controller      int              `json:"Controller"`
		DetailedStatus  []DetailedStatus `json:"Detailed Status,omitempty"`
	}

	DetailedStatus struct {
		VD          any     `json:"VD"` // Any as it can be a string or an int
		Operation   string  `json:"Operation"`
		Status      string  `json:"Status"`
		ErrCd       int     `json:"ErrCd"`
		ErrMsg      string  `json:"ErrMsg"`
		Description *string `json:"Description,omitempty"`
	}

	SystemOverview struct {
		Ctl   int    `json:"Ctl"`
		Model string `json:"Model"`
		Ports int    `json:"Ports"`
		PDs   int    `json:"PDs"`
		DGs   int    `json:"DGs"`
		DNOpt int    `json:"DNOpt"`
		VDs   int    `json:"VDs"`
		VNOpt int    `json:"VNOpt"`
		Bbu   string `json:"BBU"`
		SPR   string `json:"sPR"`
		Ds    string `json:"DS"`
		Ehs   string `json:"EHS"`
		ASOs  int    `json:"ASOs"`
		Hlth  string `json:"Hlth"`
	}

	Basics struct {
		Controller                int    `json:"Controller"`
		Model                     string `json:"Model"`
		SerialNumber              string `json:"Serial Number"`
		CurrentControllerDateTime string `json:"Current Controller Date/Time"`
		CurrentSystemDateTime     string `json:"Current System Date/time"`
		SASAddress                string `json:"SAS Address"`
		PCIAddress                string `json:"PCI Address"`
		MfgDate                   string `json:"Mfg Date"`
		ReworkDate                string `json:"Rework Date"`
		RevisionNo                string `json:"Revision No"`
	}

	// Physical Drive.
	PD struct {
		EIDSlot             string `json:"EID:Slt"`
		DeviceID            int    `json:"DID"`
		State               string `json:"State"`
		DeviceGroup         int    `json:"DG"`
		Size                string `json:"Size"` // Size (humanized)
		Interface           string `json:"Intf"`
		MediaType           string `json:"Med"`
		SelfEncryptingDrive string `json:"SED"`
		ProtectionInfo      string `json:"PI"`
		SectorSize          string `json:"SeSz"`
		Model               string `json:"Model"`
		Spun                string `json:"Sp"`
		Type                string `json:"Type"`
	}

	DriveDeviceAttributes struct {
		SerialNumber          string `json:"SN"` // Serial Number
		ManufacturerID        string `json:"Manufacturer Id"`
		ModelNumber           string `json:"Model Number"`
		NANDVendor            string `json:"NAND Vendor"`
		WWN                   string `json:"WWN"`
		FirmwareRevision      string `json:"Firmware Revision"`
		FirmwareReleaseNumber string `json:"Firmware Release Number"`
		RawSize               string `json:"Raw size"`
		CoercedSize           string `json:"Coerced size"`
		NonCoercedSize        string `json:"Non Coerced size"`
		DeviceSpeed           string `json:"Device Speed"`
		LinkSpeed             string `json:"Link Speed"`
		WriteCache            string `json:"Write Cache"`
		LogicalSectorSize     string `json:"Logical Sector Size"`
		PhysicalSectorSize    string `json:"Physical Sector Size"`
		ConnectorName         string `json:"Connector Name"`
	}

	// Virtual Drive.
	VD struct {
		DGVD                      string `json:"DG/VD"`  // Drive Group/Virtual Drive
		Type                      string `json:"TYPE"`   // Type of RAID
		State                     string `json:"State"`  // State
		Access                    string `json:"Access"` // Access Rights
		Consistent                string `json:"Consist"`
		Cache                     string `json:"Cache"` // Cache Options
		CacheCade                 string `json:"Cac"`
		ScheduledCheckConsistency string `json:"sCC"`
		Size                      string `json:"Size"` // Size (humanized)
		Name                      string `json:"Name"`
	}

	VDProperties struct {
		StripSize                string `json:"Strip Size"`
		NumberOfBlocks           int64  `json:"Number of Blocks"`
		VDHasEmulatedPD          string `json:"VD has Emulated PD"`
		SpanDepth                int    `json:"Span Depth"`
		NumberOfDrivesPerSpan    int    `json:"Number of Drives Per Span"`
		WriteCacheInitialSetting string `json:"Write Cache(initial setting)"`
		DiskCachePolicy          string `json:"Disk Cache Policy"`
		Encryption               string `json:"Encryption"`
		DataProtection           string `json:"Data Protection"`
		ActiveOperations         string `json:"Active Operations"`
		ExposedToOS              string `json:"Exposed to OS"`
		OSDriveName              string `json:"OS Drive Name"`
		CreationDate             string `json:"Creation Date"`
		CreationTime             string `json:"Creation Time"`
		EmulationType            string `json:"Emulation type"`
		CachebypassSize          string `json:"Cachebypass size"`
		CachebypassMode          string `json:"Cachebypass Mode"`
		IsLDReadyForOSRequests   string `json:"Is LD Ready for OS Requests"`
		SCSINAAID                string `json:"SCSI NAA Id"`
		UnmapEnabled             string `json:"Unmap Enabled"`
	}
)

func (c *CmdOutput) GetResponseDataByCtrlID(ctrlID int) (json.RawMessage, error) {
	for _, controller := range c.Controllers {
		if controller.CommandStatus.Controller == ctrlID {
			return controller.ResponseData, nil
		}
	}

	return nil, ErrControllerNotFound
}

// searchForKey searches for a key in a JSON object, including nested objects.
// If the key is found, the value is returned as json.RawMessage.
// The function recurses if the value is an object.
// If the key is not found, the function returns false.
func searchForKey(data json.RawMessage, targetKey string) (json.RawMessage, bool) {
	decoder := json.NewDecoder(bytes.NewReader(data))

	// Ensure we start processing as an object
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, false
	}

	for decoder.More() {
		// Read the key
		key, ok := getKey(decoder)
		if !ok {
			return nil, false
		}

		// If the key matches, return the corresponding value
		if key == targetKey {
			return getValue(decoder)
		}

		// Check if the value is an object to recurse
		var rawValue json.RawMessage
		if err := decoder.Decode(&rawValue); err != nil {
			return nil, false
		}

		// Check if it's a nested object and recurse
		if isJSONObject(rawValue) {
			// Recursively search for the key in the nested object
			if result, found := searchForKey(rawValue, targetKey); found {
				return result, true
			}
		}
	}

	return nil, false
}

// getValue reads the value from a JSON decoder.
func getValue(decoder *json.Decoder) (json.RawMessage, bool) {
	var value json.RawMessage
	if err := decoder.Decode(&value); err != nil {
		return nil, false
	}

	return value, true
}

// getKey reads the key from a JSON decoder.
func getKey(decoder *json.Decoder) (string, bool) {
	keyToken, err := decoder.Token()
	if err != nil {
		return "", false
	}

	key, ok := keyToken.(string)
	if !ok {
		return "", false
	}

	return key, true
}

// isJSONObject checks if a JSON RawMessage is an object.
func isJSONObject(data json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(data))
	return len(trimmed) > 0 && trimmed[0] == '{'
}
