package ports

type HardwareRAIDController interface {
	ControllersGetter
	PhysicalDrivesGetter
	LogicalVolumesGetter
	LogicalVolumesManager
	LVCacheSetter
	JBODSetter
	Blinker
}
