package vsphere

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/types"
)

func resourceVSphereVmfsDatastore() *schema.Resource {
	s := map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The name of the datastore.",
			Required:    true,
			ForceNew:    true,
		},
		"host": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The name of the host to set the datastore up on. This is ignored if connecting directly to ESXi, but required if not.",
			Optional:    true,
			ForceNew:    true,
		},
		"datacenter": &schema.Schema{
			Type:        schema.TypeString,
			Description: "The path to the datacenter the host is located in. This is ignored if connecting directly to ESXi. If not specified on vCenter, the default datacenter is used.",
			Optional:    true,
			ForceNew:    true,
		},
		"disks": &schema.Schema{
			Type:        schema.TypeList,
			Description: "The disks to add to the datastore.",
			Required:    true,
			MinItems:    1,
			Elem:        &schema.Schema{Type: schema.TypeString},
		},
	}
	mergeSchema(s, schemaDatastoreSummary())
	return &schema.Resource{
		Create: resourceVSphereVmfsDatastoreCreate,
		Read:   resourceVSphereVmfsDatastoreRead,
		Update: resourceVSphereVmfsDatastoreUpdate,
		Delete: resourceVSphereVmfsDatastoreDelete,
		Schema: s,
	}
}

func resourceVSphereVmfsDatastoreCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	host := d.Get("host").(string)
	datacenter := d.Get("datacenter").(string)
	dss, err := hostDatastoreSystemFromName(client, host, datacenter)
	if err != nil {
		return fmt.Errorf("error loading host datastore system: %s", err)
	}

	// To ensure the datastore is fully created with all the disks that we want
	// to add to it, first we add the initial disk, then we expand the disk with
	// the rest of the extents.
	disks := d.Get("disks").([]interface{})
	disk := disks[0].(string)
	spec, err := diskSpecForCreate(dss, disk)
	if err != nil {
		return err
	}
	spec.Vmfs.VolumeName = d.Get("name").(string)
	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	ds, err := dss.CreateVmfsDatastore(ctx, *spec)
	if err != nil {
		return fmt.Errorf("error creating datastore with disk %s: %s", disk, err)
	}

	// Now add any remaining disks.
	for _, disk := range disks[1:] {
		spec, err := diskSpecForExtend(dss, ds, disk.(string))
		if err != nil {
			// We have to destroy the created datastore here.
			if remErr := removeDatastore(dss, ds); remErr != nil {
				// We could not destroy the created datastore and there is now a dangling
				// resource. We need to instruct the user to remove the datastore
				// manually.
				return fmt.Errorf(formatCreateRollbackErrorUpdate, disk, err, remErr)
			}
			return fmt.Errorf("error extending datastore with disk %s: %s", disk, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
		defer cancel()
		if _, err := extendVmfsDatastore(ctx, dss, ds, *spec); err != nil {
			if remErr := removeDatastore(dss, ds); remErr != nil {
				// We could not destroy the created datastore and there is now a dangling
				// resource. We need to instruct the user to remove the datastore
				// manually.
				return fmt.Errorf(formatCreateRollbackErrorUpdate, disk, err, remErr)
			}
			return fmt.Errorf("error extending datastore with disk %s: %s", disk, err)
		}
	}

	props, err := datastoreProperties(ds)
	if err != nil {
		if remErr := removeDatastore(dss, ds); remErr != nil {
			// We could not destroy the created datastore and there is now a dangling
			// resource. We need to instruct the user to remove the datastore
			// manually.
			return fmt.Errorf(formatCreateRollbackErrorProperties, err, remErr)
		}
		return fmt.Errorf("could not get datastore properties after creation: %s", err)
	}
	d.SetId(props.Summary.Name)

	// Done
	return resourceVSphereVmfsDatastoreRead(d, meta)
}

func resourceVSphereVmfsDatastoreRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	name := d.Id()
	datacenter := d.Get("datacenter").(string)
	ds, err := datastoreFromName(client, name, datacenter)
	if err != nil {
		return fmt.Errorf("cannot find datastore %s: %s", name, err)
	}
	props, err := datastoreProperties(ds)
	if err != nil {
		return fmt.Errorf("could not get properties for datastore %s: %s", name, err)
	}
	if err := flattenDatastoreSummary(d, &props.Summary); err != nil {
		return err
	}

	// We also need to update the disk list from the summary.
	var disks []string
	for _, disk := range props.Info.(*types.VmfsDatastoreInfo).Vmfs.Extent {
		disks = append(disks, disk.DiskName)
	}
	if err := d.Set("disks", disks); err != nil {
		return err
	}
	return nil
}

func resourceVSphereVmfsDatastoreUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	host := d.Get("host").(string)
	name := d.Id()
	datacenter := d.Get("datacenter").(string)
	dss, err := hostDatastoreSystemFromName(client, host, datacenter)
	if err != nil {
		return fmt.Errorf("error loading host datastore system: %s", err)
	}
	ds, err := datastoreFromName(client, name, datacenter)
	if err != nil {
		return fmt.Errorf("cannot find datastore %s: %s", name, err)
	}

	// Veto this update if it means a disk was removed. Shrinking
	// datastores/removing extents is not supported.
	old, new := d.GetChange("disks")
	for _, v1 := range old.([]interface{}) {
		var found bool
		for _, v2 := range new.([]interface{}) {
			if v1.(string) == v2.(string) {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("disk %s found in state but not config (removal of disks is not supported)", v1)
		}
	}

	// Maintain a copy of the disks that have been added just in case the update
	// fails in the middle.
	old2 := make([]interface{}, len(old.([]interface{})))
	copy(old2, old.([]interface{}))

	// Now we basically reverse what we did above when we were checking for
	// removed disks, and add any new disks that have been added.
	for _, v1 := range new.([]interface{}) {
		var found bool
		for _, v2 := range old.([]interface{}) {
			if v1.(string) == v2.(string) {
				found = true
			}
		}
		if !found {
			// Add the disk
			spec, err := diskSpecForExtend(dss, ds, v1.(string))
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
			defer cancel()
			if _, err := extendVmfsDatastore(ctx, dss, ds, *spec); err != nil {
				return err
			}
			// The add was successful. Since this update is not atomic, we need to at
			// least update the disk list to make sure we don't attempt to add the
			// same disk twice.
			old2 = append(old2, v1)
			if err := d.Set("disks", old2); err != nil {
				return fmt.Errorf(formatUpdateInconsistentState, v1.(string), err)
			}
		}
	}

	// Should be done with the update here.
	return resourceVSphereVmfsDatastoreRead(d, meta)
}

func resourceVSphereVmfsDatastoreDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	host := d.Get("host").(string)
	name := d.Id()
	datacenter := d.Get("datacenter").(string)
	dss, err := hostDatastoreSystemFromName(client, host, datacenter)
	if err != nil {
		return fmt.Errorf("error loading host datastore system: %s", err)
	}
	ds, err := datastoreFromName(client, name, datacenter)
	if err != nil {
		return fmt.Errorf("cannot find datastore %s: %s", name, err)
	}

	if err := removeDatastore(dss, ds); err != nil {
		return fmt.Errorf("could not delete datastore: %s", err)
	}

	return nil
}

// formatCreateRollbackError defines the verbose error for extending a disk on
// creation where rollback is not possible.
const formatCreateRollbackErrorUpdate = `
WARNING: Dangling resource!
There was an error extending your datastore with disk: %s:
%s
Additionally, there was an error removing the created datastore:
%s
You will need to remove this datastore manually before trying again.
`

// formatCreateRollbackError defines the verbose error for extending a disk on
// creation where rollback is not possible.
const formatCreateRollbackErrorProperties = `
WARNING: Dangling resource!
After creating the datastore, there was an error fetching its properties:
%s
Additionally, there was an error removing the created datastore:
%s
You will need to remove this datastore manually before trying again.
`

// formatUpdateInconsistentState defines the verbose error when the setting
// state failed in the middle of an update opeartion. This is an error that
// will require repair of the state before TF can continue.
const formatUpdateInconsistentState = `
WARNING: Inconsistent state!
Terraform was able to add disk %s, but could not save the update to the state.
The error was:
%s
This is more than likely a bug. Please report it at:
https://github.com/terraform-providers/terraform-provider-vsphere/issues
You will also need to repair your state before trying again.
`
