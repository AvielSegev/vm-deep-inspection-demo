package vms

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	vimtypes "github.com/vmware/govmomi/vim25/types"
)

// GetVMByName retrieves a single VM by its name with full details
func (s *Service) GetVMByName(ctx context.Context, name string) (*VMDetailedResult, error) {
	s.logger.WithField("name", name).Info("Getting VM by name")

	// Find VM by name
	vm, datacenter, err := s.findVMByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("VM with name '%s' not found: %w", name, err)
	}

	// Get govmomi client for property collector
	client, err := s.getGovmomiClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get vSphere client: %w", err)
	}

	// Retrieve VM properties with comprehensive details
	var vmProp mo.VirtualMachine
	pc := property.DefaultCollector(client.Client)
	err = pc.RetrieveOne(ctx, vm.Reference(), []string{
		// Basic
		"name",
		"config.uuid",
		"config.instanceUuid",
		"config.guestFullName",
		"config.guestId",
		"config.annotation",
		"config.template",

		// Hardware
		"config.hardware.numCPU",
		"config.hardware.numCoresPerSocket",
		"config.hardware.memoryMB",
		"config.hardware.device",
		"config.version",
		"config.firmware",
		"config.cpuHotAddEnabled",
		"config.cpuHotRemoveEnabled",
		"config.memoryHotAddEnabled",
		"config.changeTrackingEnabled",

		// Runtime
		"runtime.powerState",
		"runtime.host",
		"runtime.connectionState",
		"runtime.bootTime",
		"runtime.maxCpuUsage",
		"runtime.maxMemoryUsage",
		"runtime.consolidationNeeded",
		"runtime.faultToleranceState",

		// Guest
		"guest.toolsStatus",
		"guest.toolsVersion",
		"guest.toolsRunningStatus",
		"guest.ipAddress",
		"guest.hostName",
		"guest.net",
		"guest.guestState",
		"guestHeartbeatStatus",

		// Storage
		"datastore",
		"summary.storage",
		"layoutEx.file",
		"config.files.vmPathName",

		// Network
		"network",

		// Resource allocation
		"config.cpuAllocation",
		"config.memoryAllocation",
		"resourcePool",

		// Snapshots
		"snapshot",

		// Location
		"parent",
	}, &vmProp)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve VM properties: %w", err)
	}

	// Convert to VMDetailedInfo
	vmInfo := s.convertToVMDetailedInfo(vmProp)

	s.logger.Info("VM retrieval completed")

	return &VMDetailedResult{
		Datacenter: datacenter.Name(),
		VM:         *vmInfo,
	}, nil
}

// GetVMByUUID retrieves a single VM by its UUID
func (s *Service) GetVMByUUID(ctx context.Context, uuid string) (*VMResult, error) {
	s.logger.WithField("uuid", uuid).Info("Getting VM by UUID")

	// Get govmomi client
	client, err := s.getGovmomiClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get vSphere client: %w", err)
	}

	// Create finder for object discovery
	finder := find.NewFinder(client.Client, true)

	// Get default datacenter
	datacenter, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		return nil, fmt.Errorf("no default datacenter found: %w", err)
	}
	finder.SetDatacenter(datacenter)

	// Use SearchIndex to find VM by UUID (fastest method)
	searchIndex := object.NewSearchIndex(client.Client)
	vmRef, err := searchIndex.FindByUuid(ctx, datacenter, uuid, true, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search for VM with UUID '%s': %w", uuid, err)
	}
	if vmRef == nil {
		return nil, fmt.Errorf("VM with UUID '%s' not found", uuid)
	}

	// Retrieve VM properties
	var vmProp mo.VirtualMachine
	pc := property.DefaultCollector(client.Client)
	err = pc.RetrieveOne(ctx, vmRef.Reference(), []string{
		"name",
		"config.uuid",
		"runtime.powerState",
	}, &vmProp)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve VM properties: %w", err)
	}

	// Convert to VMInfo
	vmInfo := s.convertToVMInfo(vmProp)

	s.logger.Info("VM retrieval completed")

	return &VMResult{
		Datacenter: datacenter.Name(),
		VM:         *vmInfo,
	}, nil
}

// ListVMs retrieves all virtual machines with optional filtering
func (s *Service) ListVMs(ctx context.Context, filter VMFilter) (*VMListResult, error) {
	s.logger.WithFields(logrus.Fields{
		"filter": filter,
	}).Info("Starting VM discovery")

	// Get govmomi client
	client, err := s.getGovmomiClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get vSphere client: %w", err)
	}

	// Create finder for object discovery
	finder := find.NewFinder(client.Client, true)

	// Set datacenter if specified in filter
	var datacenter *object.Datacenter
	if filter.Datacenter != "" {
		datacenter, err = finder.Datacenter(ctx, filter.Datacenter)
		if err != nil {
			return nil, fmt.Errorf("datacenter '%s' not found: %w", filter.Datacenter, err)
		}
		finder.SetDatacenter(datacenter)
	} else {
		// If no datacenter specified, use default (first one found)
		datacenter, err = finder.DefaultDatacenter(ctx)
		if err != nil {
			return nil, fmt.Errorf("no default datacenter found: %w", err)
		}
		finder.SetDatacenter(datacenter)
	}

	// Find all VMs or filter by cluster
	var vms []*object.VirtualMachine
	if filter.Cluster != "" {
		// Find VMs in specific cluster
		cluster, err := finder.ClusterComputeResource(ctx, filter.Cluster)
		if err != nil {
			return nil, fmt.Errorf("cluster '%s' not found: %w", filter.Cluster, err)
		}

		vms, err = finder.VirtualMachineList(ctx, cluster.InventoryPath+"/*")
		if err != nil {
			return nil, fmt.Errorf("failed to list VMs in cluster '%s': %w", filter.Cluster, err)
		}
	} else {
		// Find all VMs in datacenter
		vms, err = finder.VirtualMachineList(ctx, "*")
		if err != nil {
			return nil, fmt.Errorf("failed to list VMs: %w", err)
		}
	}

	s.logger.WithField("vm_count", len(vms)).Info("Found VMs in vSphere")

	// Collect VM managed object references
	var vmRefs []vimtypes.ManagedObjectReference
	for _, vm := range vms {
		vmRefs = append(vmRefs, vm.Reference())
	}

	if len(vmRefs) == 0 {
		return &VMListResult{
			Datacenter: datacenter.Name(),
			VMs:        []VMInfo{},
			Total:      0,
		}, nil
	}

	// Define properties to retrieve for all VMs
	var vmProperties []mo.VirtualMachine
	pc := property.DefaultCollector(client.Client)

	err = pc.Retrieve(ctx, vmRefs, []string{
		"name",
		"config.uuid",
		"runtime.powerState",
	}, &vmProperties)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve VM properties: %w", err)
	}

	// Convert all VMs and apply filters
	var vmInfos []VMInfo
	for _, vmProp := range vmProperties {
		vmInfo := s.convertToVMInfo(vmProp)

		// Apply name filter (contains)
		if filter.Name != "" && !strings.Contains(strings.ToLower(vmInfo.Name), strings.ToLower(filter.Name)) {
			continue
		}

		// Apply power state filter
		if filter.PowerState != "" && vmInfo.PowerState != filter.PowerState {
			continue
		}

		vmInfos = append(vmInfos, *vmInfo)
	}

	s.logger.WithField("total_vms", len(vmInfos)).Info("VM discovery completed")

	return &VMListResult{
		Datacenter: datacenter.Name(),
		VMs:        vmInfos,
		Total:      len(vmInfos),
	}, nil
}

// DeleteVM deletes a VM
func (s *Service) DeleteVM(ctx context.Context, vmName string) error {
	s.logger.WithField("vm_name", vmName).Info("Deleting VM")

	// Find VM
	vm, _, err := s.findVMByName(ctx, vmName)
	if err != nil {
		return err
	}

	// Destroy VM task
	task, err := vm.Destroy(ctx)
	if err != nil {
		return fmt.Errorf("failed to create delete task: %w", err)
	}

	s.logger.WithField("task_id", task.Reference().Value).Info("Delete task created, waiting for completion")

	// Wait for task to complete
	err = task.Wait(ctx)
	if err != nil {
		return fmt.Errorf("VM deletion failed: %w", err)
	}

	s.logger.Info("VM deleted successfully")
	return nil
}
