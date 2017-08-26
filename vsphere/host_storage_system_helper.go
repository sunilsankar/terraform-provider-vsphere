package vsphere

import (
	"context"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
)

// hostStorageSystemFromName locates a HostStorageSystem from a specified
// host name and datacenter. The default host system is used if the client is
// connected to an ESXi host, versus vCenter.
func hostStorageSystemFromName(client *govmomi.Client, host, datacenter string) (*object.HostStorageSystem, error) {
	hs, err := hostSystemOrDefault(client, host, datacenter)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	return hs.ConfigManager().StorageSystem(ctx)
}
