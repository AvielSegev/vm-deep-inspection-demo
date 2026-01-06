package vms

import (
	"context"
	"fmt"

	"github.com/kubev2v/vm-migration-detective/pkg/types"
	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	vimtypes "github.com/vmware/govmomi/vim25/types"
)

// GetSnapshotDiskInfo gets the VM moref, snapshot moref and disk path for a VM snapshot
// This is used by the inspection system to access snapshot disks via VDDK
func (s *Service) GetSnapshotDiskInfo(ctx context.Context, vmName string, snapshotName string) (*types.SnapshotDiskInfo, error) {
	s.logger.WithFields(logrus.Fields{
		"vm_name":       vmName,
		"snapshot_name": snapshotName,
	}).Debug("Getting snapshot disk info for inspection")

	// Find VM by name
	vm, _, err := s.findVMByName(ctx, vmName)
	if err != nil {
		return nil, err
	}

	// Get the VM managed object reference value
	vmMoref := vm.Reference().Value

	// Get VM properties including snapshots and disk config
	client, err := s.getGovmomiClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get vSphere client: %w", err)
	}

	var vmMo mo.VirtualMachine
	pc := property.DefaultCollector(client.Client)
	err = pc.RetrieveOne(ctx, vm.Reference(), []string{"snapshot", "config.hardware.device", "runtime.host"}, &vmMo)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM properties: %w", err)
	}

	// Check if VM has snapshots
	if vmMo.Snapshot == nil {
		return nil, fmt.Errorf("VM '%s' has no snapshots", vmName)
	}

	// Find the snapshot by name
	snapshotRef, err := s.findSnapshotInTree(vmMo.Snapshot.RootSnapshotList, snapshotName)
	if err != nil {
		return nil, fmt.Errorf("failed to find snapshot '%s': %w", snapshotName, err)
	}

	// Get snapshot moref
	snapshotMoref := snapshotRef.Snapshot.Value

	// Get disk paths from ALL virtual disks (not just the first one)
	// Use ParentFile (backing.Parent.FileName) if available
	// This is the base/parent disk file that the snapshot was created from
	var diskPaths []string
	var baseDiskPaths []string

	for _, device := range vmMo.Config.Hardware.Device {
		if disk, ok := device.(*vimtypes.VirtualDisk); ok {
			if backing, ok := disk.Backing.(*vimtypes.VirtualDiskFlatVer2BackingInfo); ok {
				diskPath := backing.FileName
				diskPaths = append(diskPaths, diskPath)

				// Check if backing has a Parent
				// Parent points to the base disk file that the snapshot was created from
				var baseDiskPath string
				if backing.Parent != nil && backing.Parent.FileName != "" {
					baseDiskPath = backing.Parent.FileName
					s.logger.WithFields(logrus.Fields{
						"disk_path":   diskPath,
						"parent_file": baseDiskPath,
					}).Debug("Found parent file from disk backing")
				} else {
					// Fallback: calculate base disk path (remove delta disk suffix like -000002)
					baseDiskPath = s.getBaseDiskPath(diskPath)
					s.logger.WithFields(logrus.Fields{
						"disk_path":       diskPath,
						"calculated_base": baseDiskPath,
					}).Debug("Calculated base disk path (no parent in backing)")
				}
				baseDiskPaths = append(baseDiskPaths, baseDiskPath)
			}
		}
	}

	if len(diskPaths) == 0 {
		return nil, fmt.Errorf("no disks found for VM '%s'", vmName)
	}

	if len(baseDiskPaths) == 0 {
		return nil, fmt.Errorf("no base disk paths found for VM '%s'", vmName)
	}

	// Get compute resource path (host/cluster) for vpx:// URL
	var computeResourcePath string
	if vmMo.Runtime.Host != nil {
		finder := find.NewFinder(client.Client, true)
		host, err := finder.ObjectReference(ctx, *vmMo.Runtime.Host)
		if err == nil {
			if hostObj, ok := host.(*object.HostSystem); ok {
				// Get the host's inventory path
				computeResourcePath = hostObj.InventoryPath
				s.logger.WithField("compute_resource_path", computeResourcePath).Debug("Got compute resource path from host")
			}
		}
		// If we couldn't get the host path, try to get it from the host's parent (cluster)
		if computeResourcePath == "" && vmMo.Runtime.Host != nil {
			// Try to get cluster path by finding the host's parent
			var hostMo mo.HostSystem
			err = pc.RetrieveOne(ctx, *vmMo.Runtime.Host, []string{"parent"}, &hostMo)
			if err == nil && hostMo.Parent != nil {
				parentObj, err := finder.ObjectReference(ctx, *hostMo.Parent)
				if err == nil {
					if clusterObj, ok := parentObj.(*object.ClusterComputeResource); ok {
						computeResourcePath = clusterObj.InventoryPath
						s.logger.WithField("compute_resource_path", computeResourcePath).Debug("Got compute resource path from cluster")
					} else if computeResourceObj, ok := parentObj.(*object.ComputeResource); ok {
						computeResourcePath = computeResourceObj.InventoryPath
						s.logger.WithField("compute_resource_path", computeResourcePath).Debug("Got compute resource path from compute resource")
					}
				}
			}
		}
	}

	if computeResourcePath == "" {
		return nil, fmt.Errorf("failed to get compute resource path for VM '%s'", vmName)
	}

	s.logger.WithFields(logrus.Fields{
		"vm_moref":              vmMoref,
		"snapshot_moref":        snapshotMoref,
		"disk_count":            len(diskPaths),
		"disk_paths":            diskPaths,
		"base_disk_paths":       baseDiskPaths,
		"compute_resource_path": computeResourcePath,
	}).Debug("Got snapshot disk info")

	return &types.SnapshotDiskInfo{
		VMMoref:             vmMoref,
		SnapshotMoref:       snapshotMoref,
		DiskPaths:           diskPaths,
		BaseDiskPaths:       baseDiskPaths,
		ComputeResourcePath: computeResourcePath,
	}, nil
}

// FindSnapshotByName finds a snapshot by name on a VM
func (s *Service) FindSnapshotByName(ctx context.Context, vmName string, snapshotName string) (*vimtypes.ManagedObjectReference, error) {
	s.logger.WithFields(logrus.Fields{
		"vm_name":       vmName,
		"snapshot_name": snapshotName,
	}).Info("Finding snapshot by name")

	// Find VM by name
	vm, _, err := s.findVMByName(ctx, vmName)
	if err != nil {
		return nil, err
	}

	// Get snapshot tree
	var vmProps mo.VirtualMachine
	client, err := s.getGovmomiClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get vSphere client: %w", err)
	}

	pc := property.DefaultCollector(client.Client)
	err = pc.RetrieveOne(ctx, vm.Reference(), []string{"snapshot"}, &vmProps)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve VM snapshots: %w", err)
	}

	if vmProps.Snapshot == nil {
		return nil, fmt.Errorf("VM '%s' has no snapshots", vmName)
	}

	// Search for snapshot by name
	var findSnapshot func(tree []vimtypes.VirtualMachineSnapshotTree) *vimtypes.ManagedObjectReference
	findSnapshot = func(tree []vimtypes.VirtualMachineSnapshotTree) *vimtypes.ManagedObjectReference {
		for _, node := range tree {
			if node.Name == snapshotName {
				return &node.Snapshot
			}
			if len(node.ChildSnapshotList) > 0 {
				if ref := findSnapshot(node.ChildSnapshotList); ref != nil {
					return ref
				}
			}
		}
		return nil
	}

	snapshotRef := findSnapshot(vmProps.Snapshot.RootSnapshotList)
	if snapshotRef == nil {
		return nil, fmt.Errorf("snapshot '%s' not found on VM '%s'", snapshotName, vmName)
	}

	s.logger.Info("Snapshot found successfully")
	return snapshotRef, nil
}

// CreateSnapshot creates a snapshot for a VM
func (s *Service) CreateSnapshot(ctx context.Context, vmName string, snapshotName string, description string, memory bool, quiesce bool) (string, error) {
	s.logger.WithFields(logrus.Fields{
		"vm_name":       vmName,
		"snapshot_name": snapshotName,
		"memory":        memory,
		"quiesce":       quiesce,
	}).Info("Creating VM snapshot")

	// Find VM by name using the helper function
	vm, _, err := s.findVMByName(ctx, vmName)
	if err != nil {
		return "", err
	}

	// Create snapshot task
	task, err := vm.CreateSnapshot(ctx, snapshotName, description, memory, quiesce)
	if err != nil {
		return "", fmt.Errorf("failed to create snapshot task: %w", err)
	}

	s.logger.WithField("task_id", task.Reference().Value).Info("Snapshot task created, waiting for completion")

	// Wait for task to complete
	err = task.Wait(ctx)
	if err != nil {
		return "", fmt.Errorf("snapshot creation failed: %w", err)
	}

	s.logger.Info("Snapshot created successfully")

	// Return the task reference as snapshot ID
	return task.Reference().Value, nil
}

// findSnapshotInTree recursively searches for a snapshot by name in the snapshot tree
func (s *Service) findSnapshotInTree(snapshots []vimtypes.VirtualMachineSnapshotTree, name string) (*vimtypes.VirtualMachineSnapshotTree, error) {
	for idx := range snapshots {
		if snapshots[idx].Name == name {
			return &snapshots[idx], nil
		}
		// Search in child snapshots
		if len(snapshots[idx].ChildSnapshotList) > 0 {
			result, err := s.findSnapshotInTree(snapshots[idx].ChildSnapshotList, name)
			if err == nil {
				return result, nil
			}
		}
	}
	return nil, fmt.Errorf("snapshot '%s' not found", name)
}

// getBaseDiskPath removes the -XXXXXX delta disk suffix to get the base VMDK path
// Example: "[datastore] vm/vm-000002.vmdk" -> "[datastore] vm/vm.vmdk"
func (s *Service) getBaseDiskPath(diskPath string) string {
	// Find the last occurrence of .vmdk
	vmdkIndex := len(diskPath) - len(".vmdk")
	if vmdkIndex < 0 || diskPath[vmdkIndex:] != ".vmdk" {
		// Not a .vmdk file, return as-is
		return diskPath
	}

	// Find the part before .vmdk
	prefix := diskPath[:vmdkIndex]

	// Look for -XXXXXX pattern (6 digits) before .vmdk
	// Example: "vm-000002" -> "vm"
	if len(prefix) >= 7 && prefix[len(prefix)-7] == '-' {
		// Check if last 6 characters are digits
		isAllDigits := true
		for i := len(prefix) - 6; i < len(prefix); i++ {
			if prefix[i] < '0' || prefix[i] > '9' {
				isAllDigits = false
				break
			}
		}
		if isAllDigits {
			// Remove -XXXXXX suffix
			return prefix[:len(prefix)-7] + ".vmdk"
		}
	}

	// No delta disk suffix found, return original path
	return diskPath
}

// RemoveSnapshot deletes a snapshot and snapshot children by name from a VM
func (s *Service) RemoveSnapshot(
	ctx context.Context,
	vmName string,
	snapshotName string,
	consolidate *bool,
) (string, error) {
	s.logger.WithFields(logrus.Fields{
		"vm_name":         vmName,
		"snapshot_name":   snapshotName,
		"remove_children": true,
	}).Info("Deleting VM snapshot")

	// Find VM
	vm, _, err := s.findVMByName(ctx, vmName)
	if err != nil {
		return "", err
	}

	task, err := vm.RemoveSnapshot(ctx, snapshotName, true, consolidate)
	if err != nil {
		fmt.Printf(err.Error())
		return "", fmt.Errorf("failed to initiate delete snapshot task: %w", err)
	}

	// Create snapshot task
	s.logger.WithField("task_id", task.Reference().Value).Info("Snapshot task created, waiting for completion")

	// Wait for task to complete
	err = task.Wait(ctx)
	if err != nil {
		return "", fmt.Errorf("snapshot deletion failed: %w", err)
	}

	s.logger.Info("Snapshot deleted successfully")

	// Return the task reference as snapshot ID
	return task.Reference().Value, nil
}
