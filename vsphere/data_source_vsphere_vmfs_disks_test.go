package vsphere

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccDataSourceVSphereVmfsDisks(t *testing.T) {
	var tp *testing.T
	testAccDataSourceVSphereVmfsDisksCases := []struct {
		name     string
		testCase resource.TestCase
	}{
		{
			"basic",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccDataSourceVSphereVmfsDisksPreCheck(tp)
				},
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: testAccDataSourceVSphereVmfsDisksConfig(),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckOutput("found", "true"),
						),
					},
				},
			},
		},
	}

	for _, tc := range testAccDataSourceVSphereVmfsDisksCases {
		t.Run(tc.name, func(t *testing.T) {
			tp = t
			resource.Test(t, tc.testCase)
		})
	}
}

func testAccDataSourceVSphereVmfsDisksPreCheck(t *testing.T) {
	if os.Getenv("VSPHERE_ESXI_HOST") == "" {
		t.Skip("set VSPHERE_ESXI_HOST to run vsphere_vmfs_disks acceptance tests")
	}
	if os.Getenv("VSPHERE_VMFS_EXPECTED") == "" {
		t.Skip("set VSPHERE_ESXI_HOST to run vsphere_vmfs_disks acceptance tests")
	}
}

func testAccDataSourceVSphereVmfsDisksConfig() string {
	return fmt.Sprintf(`
variable "esxi_host" {
  type    = "string"
  default = "%s"
}

variable "datacenter" {
  type    = "string"
  default = "%s"
}

data "vsphere_vmfs_disks" "available" {
  host       = "${var.esxi_host}"
  datacenter = "${var.datacenter}"
}

output "found" {
  value = "${contains(data.vsphere_vmfs_disks.available.disks, "%s")}"
}
`, os.Getenv("VSPHERE_ESXI_HOST"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_VMFS_EXPECTED"))
}
