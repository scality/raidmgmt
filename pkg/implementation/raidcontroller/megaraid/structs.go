package megaraid

import (
	"encoding/json"
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
		EIDSlot  string `json:"EID:Slt"`
		DeviceID int    `json:"DID"`
		State    string `json:"State"`
		// DG can be either a "-" or a int, since we are
		// not using it today let's just comment it out
		// DeviceGroup         int    `json:"DG"`
		Size                string `json:"Size"` // Size (humanized)
		Interface           string `json:"Intf"`
		MediaType           string `json:"Med"` // Media Type (HDD, SSD, NVMe)
		SelfEncryptingDrive string `json:"SED"`
		ProtectionInfo      string `json:"PI"`
		SectorSize          string `json:"SeSz"`
		Model               string `json:"Model"`
		Spun                string `json:"Sp"`
		Type                string `json:"Type"` // Type of disk (JBOD, RAID)
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

	SupportedAdapterOperations struct {
		RebuildRate                               string `json:"Rebuild Rate"`
		CCRate                                    string `json:"CC Rate"`
		BGIRate                                   string `json:"BGI Rate "`
		ReconstructRate                           string `json:"Reconstruct Rate"`
		PatrolReadRate                            string `json:"Patrol Read Rate"`
		AlarmControl                              string `json:"Alarm Control"`
		ClusterSupport                            string `json:"Cluster Support"`
		Bbu                                       string `json:"BBU"`
		Spanning                                  string `json:"Spanning"`
		DedicatedHotSpare                         string `json:"Dedicated Hot Spare"`
		RevertibleHotSpares                       string `json:"Revertible Hot Spares"`
		ForeignConfigImport                       string `json:"Foreign Config Import"`
		SelfDiagnostic                            string `json:"Self Diagnostic"`
		AllowMixedRedundancyOnArray               string `json:"Allow Mixed Redundancy on Array"`
		GlobalHotSpares                           string `json:"Global Hot Spares"`
		DenySCSIPassthrough                       string `json:"Deny SCSI Passthrough"`
		DenySMPPassthrough                        string `json:"Deny SMP Passthrough"`
		DenySTPPassthrough                        string `json:"Deny STP Passthrough"`
		SupportMoreThan8Phys                      string `json:"Support more than 8 Phys"`
		FWAndEventTimeInGMT                       string `json:"FW and Event Time in GMT"`
		SupportEnhancedForeignImport              string `json:"Support Enhanced Foreign Import"`
		SupportEnclosureEnumeration               string `json:"Support Enclosure Enumeration"`
		SupportAllowedOperations                  string `json:"Support Allowed Operations"`
		AbortCCOnError                            string `json:"Abort CC on Error"`
		SupportMultipath                          string `json:"Support Multipath"`
		SupportOddEvenDriveCountInRAID1E          string `json:"Support Odd & Even Drive count in RAID1E"`
		SupportSecurity                           string `json:"Support Security"`
		SupportConfigPageModel                    string `json:"Support Config Page Model"`
		SupportTheOCEWithoutAddingDrives          string `json:"Support the OCE without adding drives"`
		SupportEKM                                string `json:"Support EKM"`
		SnapshotEnabled                           string `json:"Snapshot Enabled"`
		SupportPFK                                string `json:"Support PFK"`
		SupportPI                                 string `json:"Support PI"`
		SupportLdBBMInfo                          string `json:"Support Ld BBM Info"`
		SupportShieldState                        string `json:"Support Shield State"`
		BlockSSDWriteDiskCacheChange              string `json:"Block SSD Write Disk Cache Change"`
		SupportSuspendResumeBGOps                 string `json:"Support Suspend Resume BG ops"`
		SupportEmergencySpares                    string `json:"Support Emergency Spares"`
		SupportSetLinkSpeed                       string `json:"Support Set Link Speed"`
		SupportBootTimePFKChange                  string `json:"Support Boot Time PFK Change"`
		SupportJBOD                               string `json:"Support JBOD"`
		DisableOnlinePFKChange                    string `json:"Disable Online PFK Change"`
		SupportPerfTuning                         string `json:"Support Perf Tuning"`
		SupportSSDPatrolRead                      string `json:"Support SSD PatrolRead"`
		RealTimeScheduler                         string `json:"Real Time Scheduler"`
		SupportResetNow                           string `json:"Support Reset Now"`
		SupportEmulatedDrives                     string `json:"Support Emulated Drives"`
		HeadlessMode                              string `json:"Headless Mode"`
		DedicatedHotSparesLimited                 string `json:"Dedicated HotSpares Limited"`
		PointInTimeProgress                       string `json:"Point In Time Progress"`
		ExtendedLD                                string `json:"Extended LD"`
		SupportUnevenSpan                         string `json:"Support Uneven span "`
		SupportConfigAutoBalance                  string `json:"Support Config Auto Balance"`
		SupportMaintenanceMode                    string `json:"Support Maintenance Mode"`
		SupportDiagnosticResults                  string `json:"Support Diagnostic results"`
		SupportExtEnclosure                       string `json:"Support Ext Enclosure"`
		SupportSesmonitoring                      string `json:"Support Sesmonitoring"`
		SupportSecurityonJBOD                     string `json:"Support SecurityonJBOD"`
		SupportForceFlash                         string `json:"Support ForceFlash"`
		SupportDisableImmediateIO                 string `json:"Support DisableImmediateIO"`
		SupportLargeIOSupport                     string `json:"Support LargeIOSupport"`
		SupportDrvActivityLEDSetting              string `json:"Support DrvActivityLEDSetting"`
		SupportFlushWriteVerify                   string `json:"Support FlushWriteVerify"`
		SupportCPLDUpdate                         string `json:"Support CPLDUpdate"`
		SupportForceTo512E                        string `json:"Support ForceTo512e"`
		SupportDiscardCacheDuringLDDelete         string `json:"Support discardCacheDuringLDDelete"`
		SupportJBODWriteCache                     string `json:"Support JBOD Write cache"`
		SupportLargeQDSupport                     string `json:"Support Large QD Support"`
		SupportCtrlInfoExtended                   string `json:"Support Ctrl Info Extended"`
		SupportIButtonLess                        string `json:"Support IButton less"`
		SupportAESEncryptionAlgorithm             string `json:"Support AES Encryption Algorithm"`
		SupportEncryptedMFC                       string `json:"Support Encrypted MFC"`
		SupportSnapdump                           string `json:"Support Snapdump"`
		SupportForcePersonalityChange             string `json:"Support Force Personality Change"`
		SupportDualFwImage                        string `json:"Support Dual Fw Image"`
		SupportPSOCUpdate                         string `json:"Support PSOC Update"`
		SupportSecureBoot                         string `json:"Support Secure Boot"`
		SupportDebugQueue                         string `json:"Support Debug Queue"`
		SupportLeastLatencyMode                   string `json:"Support Least Latency Mode"`
		SupportOnDemandSnapdump                   string `json:"Support OnDemand Snapdump"`
		SupportClearSnapdump                      string `json:"Support Clear Snapdump"`
		SupportFWTriggeredSnapdump                string `json:"Support FW Triggered Snapdump"`
		SupportPHYCurrentSpeed                    string `json:"Support PHY current speed"`
		SupportLaneCurrentSpeed                   string `json:"Support Lane current speed"`
		SupportNVMeWidth                          string `json:"Support NVMe Width"`
		SupportLaneDeviceType                     string `json:"Support Lane DeviceType"`
		SupportExtendedDrivePerformanceMonitoring string `json:"Support Extended Drive performance Monitoring"` //nolint:lll // The JSON tag is long
		SupportNVMeRepair                         string `json:"Support NVMe Repair"`
		SupportPlatformSecurity                   string `json:"Support Platform Security"`
		SupportNoneModeParams                     string `json:"Support None Mode Params"`
		SupportExtendedControllerProperty         string `json:"Support Extended Controller Property"`
		SupportSmartPollIntervalForDirectAttached string `json:"Support Smart Poll Interval for DirectAttached"` //nolint:lll // The JSON tag is long
		SupportWriteJournalPinning                string `json:"Support Write Journal Pinning"`
		SupportSMPPassthruWithPortNumber          string `json:"Support SMP Passthru with Port Number"`
		SupportNVMeInitErrorDeviceConnectorIndex  string `json:"Support NVMe Init Error Device ConnectorIndex"` //nolint:lll // The JSON tag is long
	}

	Capabilities struct {
		SupportedDrives                string `json:"Supported Drives"`
		RAIDLevelSupported             string `json:"RAID Level Supported"`
		EnableJBOD                     string `json:"Enable JBOD"`
		MixInEnclosure                 string `json:"Mix in Enclosure"`
		MixOfSASSATAOfHDDTypeInVD      string `json:"Mix of SAS/SATA of HDD type in VD"`
		MixOfSASSATAOfSSDTypeInVD      string `json:"Mix of SAS/SATA of SSD type in VD"`
		MixOfSSDHDDInVD                string `json:"Mix of SSD/HDD in VD"`
		SASDisable                     string `json:"SAS Disable"`
		MaxArmsPerVD                   int    `json:"Max Arms Per VD"`
		MaxSpansPerVD                  int    `json:"Max Spans Per VD"`
		MaxArrays                      int    `json:"Max Arrays"`
		MaxVDPerArray                  int    `json:"Max VD per array"`
		MaxNumberOfVDs                 int    `json:"Max Number of VDs"`
		MaxParallelCommands            int    `json:"Max Parallel Commands"`
		MaxSGECount                    int    `json:"Max SGE Count"`
		MaxDataTransferSize            string `json:"Max Data Transfer Size"`
		MaxStripsPerIO                 int    `json:"Max Strips PerIO"`
		MaxConfigurableCacheCadeSizeGB int    `json:"Max Configurable CacheCade Size(GB)"`
		MaxTransportableDGs            int    `json:"Max Transportable DGs"`
		EnableSnapdump                 string `json:"Enable Snapdump"`
		EnableSCSIUnmap                string `json:"Enable SCSI Unmap"`
		FDEDriveMixSupport             string `json:"FDE Drive Mix Support"`
		MinStripSize                   string `json:"Min Strip Size"`
		MaxStripSize                   string `json:"Max Strip Size"`
	}
)
