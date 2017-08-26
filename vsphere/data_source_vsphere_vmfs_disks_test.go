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
		{
			"with regular expression",
			resource.TestCase{
				PreCheck: func() {
					testAccPreCheck(tp)
					testAccDataSourceVSphereVmfsDisksPreCheck(tp)
				},
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: testAccDataSourceVSphereVmfsDisksConfigRegexp(),
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckOutput("expected_length", "true"),
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
		t.Skip("set VSPHERE_VMFS_EXPECTED to run vsphere_vmfs_disks acceptance tests")
	}
	if os.Getenv("VSPHERE_VMFS_REGEXP") == "" {
		t.Skip("set VSPHERE_VMFS_REGEXP to run vsphere_vmfs_disks acceptance tests")
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
  rescan     = true
}

output "found" {
  value = "${contains(data.vsphere_vmfs_disks.available.disks, "%s")}"
}
`, os.Getenv("VSPHERE_ESXI_HOST"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_VMFS_EXPECTED"))
}

func testAccDataSourceVSphereVmfsDisksConfigRegexp() string {
	return fmt.Sprintf(`
variable "esxi_host" {
  type    = "string"
  default = "%s"
}

variable "datacenter" {
  type    = "string"
  default = "%s"
}

variable "regexp" {
  type    = "string"
  default = "%s"
}

data "vsphere_vmfs_disks" "available" {
  host       = "${var.esxi_host}"
  datacenter = "${var.datacenter}"
  rescan     = true
  filter     = "${var.regexp}"
}

output "expected_length" {
	value = "${length(data.vsphere_vmfs_disks.available.disks) == 2 ? "true" : "false" }"
}
`, os.Getenv("VSPHERE_ESXI_HOST"), os.Getenv("VSPHERE_DATACENTER"), os.Getenv("VSPHERE_VMFS_REGEXP"))
}
