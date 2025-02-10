package physicaldriveresolver

const devDiskByIDPathFormat = "/dev/disk/by-id/%s"

type PhysicalDriveResolver interface {
	ResolvePhysicalDriveDeviceNameFromID(diskID string) (string, error)
}
