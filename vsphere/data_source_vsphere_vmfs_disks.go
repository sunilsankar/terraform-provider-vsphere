package vsphere

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/types"
)

func dataSourceVSphereVmfsDisks() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceVSphereVmfsDisksRead,

		Schema: map[string]*schema.Schema{
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The name of the host to put this virtual switch on. This is ignored if connecting directly to ESXi, but required if not.",
				Optional:    true,
			},
			"datacenter": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The path to the datacenter the host is located in. This is ignored if connecting directly to ESXi. If not specified on vCenter, the default datacenter is used.",
				Optional:    true,
			},
			"disks": &schema.Schema{
				Type:        schema.TypeList,
				Description: "The names of the disks discovered by the search.",
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceVSphereVmfsDisksRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*govmomi.Client)
	host := d.Get("host").(string)
	datacenter := d.Get("datacenter").(string)
	hss, err := hostDatastoreSystemFromName(client, host, datacenter)
	if err != nil {
		return fmt.Errorf("error loading host datastore system: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultAPITimeout)
	defer cancel()
	disks, err := hss.QueryAvailableDisksForVmfs(ctx)
	if err != nil {
		return fmt.Errorf("error querying for disks: %s", err)
	}

	d.SetId(time.Now().UTC().String())

	if saveVmfsDiskNames(disks, d); err != nil {
		return fmt.Errorf("error saving results to state: %s", err)
	}

	return nil
}

// saveVmfsDiskNames saves the CanonicalNames of the search results to the to
// the "disks" attribute in the passed in ResourceData.
func saveVmfsDiskNames(disks []types.HostScsiDisk, d *schema.ResourceData) error {
	var s []string
	for _, disk := range disks {
		s = append(s, disk.CanonicalName)
	}
	return d.Set("disks", s)
}
