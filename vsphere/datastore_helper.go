package vsphere

import (
	"fmt"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"golang.org/x/net/context"
)

// datastoreFromName locates a Datastore by name available from a specific
// datacenter. If the user is connecting over ESXi, the default datacenter is
// used.
func datastoreFromName(client *govmomi.Client, name, datacenter string) (*object.Datastore, error) {
	finder := find.NewFinder(client.Client, false)

	var err error
	dc, err := getDatacenter(client, datacenter)
	if err != nil {
		return nil, fmt.Errorf("could not get datacenter: %s", err)
	}
	finder.SetDatacenter(dc)
	if err != nil {
		return nil, fmt.Errorf("error setting datacenter in finder: %s", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	return finder.Datastore(ctx, name)
}

// datastoreProperties is a convenience method that wraps fetching the
// Datastore MO from its higher-level object.
func datastoreProperties(ds *object.Datastore) (*mo.Datastore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	var props mo.Datastore
	if err := ds.Properties(ctx, ds.Reference(), nil, &props); err != nil {
		return nil, err
	}
	return &props, nil
}
