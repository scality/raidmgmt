package ports

type SoftwareRAIDController interface {
	PhysicalDrivesGetter
	LogicalVolumesGetter
	LogicalVolumesManager
}
