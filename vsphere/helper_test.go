package vsphere

import (
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/terraform/terraform"
	"github.com/vmware/govmomi"
)

// testCheckVariables bundles common variables needed by various test checkers.
type testCheckVariables struct {
	// A client for various operations.
	client *govmomi.Client

	// The subject resource's ID.
	resourceID string

	// The ESXi host that a various API call is directed at.
	esxiHost string

	// The datacenter that a various API call is directed at.
	datacenter string

	// A timeout to pass to various context creation calls.
	timeout time.Duration
}

func testClientVariablesForResource(s *terraform.State, addr string) (testCheckVariables, error) {
	rs, ok := s.RootModule().Resources[addr]
	if !ok {
		return testCheckVariables{}, fmt.Errorf("%s not found in state", addr)
	}

	return testCheckVariables{
		client:     testAccProvider.Meta().(*govmomi.Client),
		resourceID: rs.Primary.ID,
		esxiHost:   os.Getenv("VSPHERE_ESXI_HOST"),
		datacenter: os.Getenv("VSPHERE_DATACENTER"),
		timeout:    time.Minute * 5,
	}, nil
}
