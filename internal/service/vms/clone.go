package vms

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/find"
	vimtypes "github.com/vmware/govmomi/vim25/types"
)

// CreateLinkedClone creates a linked clone from a snapshot
func (s *Service) CreateLinkedClone(ctx context.Context, vmName string, snapshotRef *vimtypes.ManagedObjectReference, cloneName string) error {
	s.logger.WithFields(logrus.Fields{
		"vm_name":    vmName,
		"clone_name": cloneName,
	}).Info("Creating linked clone from snapshot")

	// Find source VM
	vm, datacenter, err := s.findVMByName(ctx, vmName)
	if err != nil {
		return err
	}

	// Get govmomi client
	client, err := s.getGovmomiClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get vSphere client: %w", err)
	}

	// Get VM folder
	finder := find.NewFinder(client.Client, true)
	finder.SetDatacenter(datacenter)

	vmFolder, err := finder.FolderOrDefault(ctx, "vm")
	if err != nil {
		return fmt.Errorf("failed to find VM folder: %w", err)
	}

	// Create linked clone spec
	cloneSpec := vimtypes.VirtualMachineCloneSpec{
		Location: vimtypes.VirtualMachineRelocateSpec{
			DiskMoveType: string(vimtypes.VirtualMachineRelocateDiskMoveOptionsCreateNewChildDiskBacking),
		},
		Snapshot: snapshotRef,
		PowerOn:  false,
		Template: false,
	}

	// Create clone task
	task, err := vm.Clone(ctx, vmFolder, cloneName, cloneSpec)
	if err != nil {
		return fmt.Errorf("failed to create clone task: %w", err)
	}

	s.logger.WithField("task_id", task.Reference().Value).Info("Clone task created, waiting for completion")

	// Wait for task to complete
	err = task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("clone creation failed: %w", err)
	}

	s.logger.Info("Linked clone created successfully")
	return nil
}

// InspectVMFromSnapshot inspects a VM by creating a temporary clone from a snapshot
func (s *Service) InspectVMFromSnapshot(ctx context.Context, vmName string, snapshotName string, inspector interface{}) error {
	// Generate unique clone name
	cloneName := fmt.Sprintf("%s-inspect-clone-%d", vmName, time.Now().Unix())

	s.logger.WithFields(logrus.Fields{
		"vm_name":       vmName,
		"snapshot_name": snapshotName,
		"clone_name":    cloneName,
	}).Info("Starting VM inspection from snapshot")

	// Find snapshot
	snapshotRef, err := s.FindSnapshotByName(ctx, vmName, snapshotName)
	if err != nil {
		return fmt.Errorf("failed to find snapshot: %w", err)
	}

	// Create linked clone
	err = s.CreateLinkedClone(ctx, vmName, snapshotRef, cloneName)
	if err != nil {
		return fmt.Errorf("failed to create linked clone: %w", err)
	}

	// Ensure cleanup of clone
	defer func() {
		s.logger.Info("Cleaning up inspection clone")
		cleanupErr := s.DeleteVM(ctx, cloneName)
		if cleanupErr != nil {
			s.logger.WithError(cleanupErr).Error("Failed to delete inspection clone")
		}
	}()

	// Note: The actual virt-inspector execution will be handled by the API handler
	// This method just manages the clone lifecycle

	s.logger.Info("Inspection clone ready for inspection")
	return nil
}
