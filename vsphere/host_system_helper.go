package vsphere

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
)

// hostSystemOrDefault returns a HostSystem from a specific host name and
// datacenter. If the user is connecting over ESXi, the default host system is
// used.
func hostSystemOrDefault(client *govmomi.Client, host, datacenter string) (*object.HostSystem, error) {
	finder := find.NewFinder(client.Client, false)

	var hs *object.HostSystem
	var err error
	dc, err := getDatacenter(client, datacenter)
	if err != nil {
		return nil, fmt.Errorf("could not get datacenter: %s", err)
	}
	finder.SetDatacenter(dc)

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	switch t := client.ServiceContent.About.ApiType; t {
	case "HostAgent":
		hs, err = finder.DefaultHostSystem(ctx)
	case "VirtualCenter":
		hs, err = finder.HostSystem(ctx, host)
	default:
		return nil, fmt.Errorf("unsupported ApiType: %s", t)
	}
	if err != nil {
		return nil, fmt.Errorf("error loading host system: %s", err)
	}
	return hs, nil
}
