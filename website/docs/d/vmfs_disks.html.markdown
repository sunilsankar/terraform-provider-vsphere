---
layout: "vsphere"
page_title: "VMware vSphere: vsphere_vmfs_disks"
sidebar_current: "docs-vsphere-data-source-vmfs-disks"
description: |-
  A data source that can be used to discover storage devices that can be used for VMFS datastores.
---

# vsphere\_vmfs\_disks

The `vsphere_vmfs_disks` data source can be used to discover the storage
devices available on a ESXi host. This data source can be combined with the
`vsphere_vmfs_datastore` resource to create VMFS datastores based off a set of
discovered disks.

## Example Usage

```hcl
data "vsphere_vmfs_disks" "available" {
  host       = "esx1.vsphere-lab.internal"
  datacenter = "lab-dc1"
  rescan     = true
  filter     = "mpx.vmhba1:C0:T[12]:L0"
}
```

## Argument Reference

The following arguments are supported:

* `host` - (String) The host to search for devices on. Required when using
  vCenter, not required when using ESXi.
* `datacenter` - (String) The name of the datacenter the host is in. Required
  when using vCenter, not required when using ESXi.
* `rescan` - (Boolean, optional) Whether or not to rescan storage adapters
  before searching for disks. This may lengthen the time it takes to perform
  the search. Default: `false`.
* `filter` - (String, optional) A regular expression to filter the disks
  against. Only disks with canonical names that match will be included. 

~> **NOTE:** Using a `filter` is recommended if there is any chance the host
will have any specific storage devices added to it that may affect the order of
the output `disks` attribute below, which is lexicographically sorted.

## Attribute Reference

* `disks` - (List of strings) A lexicographically sorted list of devices
  discovered by the operation, matching the supplied `filter`, if provided.
