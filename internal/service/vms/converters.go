package vms

import (
	"strings"
	"time"

	"github.com/vmware/govmomi/vim25/mo"
)

// convertToVMInfo converts a vSphere VM managed object to VMInfo
func (s *Service) convertToVMInfo(vm mo.VirtualMachine) *VMInfo {
	return &VMInfo{
		UUID:       vm.Config.Uuid,
		Name:       vm.Name,
		PowerState: string(vm.Runtime.PowerState),
	}
}

// convertToVMDetailedInfo converts a vSphere VM managed object to VMDetailedInfo
func (s *Service) convertToVMDetailedInfo(vm mo.VirtualMachine) *VMDetailedInfo {
	info := &VMDetailedInfo{
		UUID:       vm.Config.Uuid,
		Name:       vm.Name,
		PowerState: string(vm.Runtime.PowerState),
	}

	// Basic Config properties
	if vm.Config != nil {
		info.InstanceUUID = vm.Config.InstanceUuid
		info.GuestFullName = vm.Config.GuestFullName
		info.GuestID = vm.Config.GuestId
		info.Version = vm.Config.Version
		info.Annotation = vm.Config.Annotation
		info.FirmwareType = vm.Config.Firmware
		info.Template = vm.Config.Template
		info.ChangeTrackingEnabled = vm.Config.ChangeTrackingEnabled != nil && *vm.Config.ChangeTrackingEnabled
		info.BiosUUID = vm.Config.Uuid
		info.CPUHotAddEnabled = vm.Config.CpuHotAddEnabled != nil && *vm.Config.CpuHotAddEnabled
		info.CPUHotRemoveEnabled = vm.Config.CpuHotRemoveEnabled != nil && *vm.Config.CpuHotRemoveEnabled
		info.MemoryHotAddEnabled = vm.Config.MemoryHotAddEnabled != nil && *vm.Config.MemoryHotAddEnabled

		// Hardware properties
		if vm.Config.Hardware.NumCPU > 0 {
			info.NumCPU = vm.Config.Hardware.NumCPU
		}
		if vm.Config.Hardware.NumCoresPerSocket > 0 {
			info.NumCoresPerSocket = vm.Config.Hardware.NumCoresPerSocket
		}
		if vm.Config.Hardware.MemoryMB > 0 {
			info.MemoryMB = vm.Config.Hardware.MemoryMB
		}

		// VM files
		if vm.Config.Files.VmPathName != "" {
			info.VMPathName = vm.Config.Files.VmPathName
		}

		// Resource allocation
		if vm.Config.CpuAllocation != nil {
			info.ResourceAllocation.CPUReservation = *vm.Config.CpuAllocation.Reservation
			if vm.Config.CpuAllocation.Limit != nil && *vm.Config.CpuAllocation.Limit != -1 {
				info.ResourceAllocation.CPULimit = *vm.Config.CpuAllocation.Limit
			}
			if vm.Config.CpuAllocation.Shares != nil {
				info.ResourceAllocation.CPUShares = vm.Config.CpuAllocation.Shares.Shares
				info.ResourceAllocation.CPUSharesLevel = string(vm.Config.CpuAllocation.Shares.Level)
			}
		}
		if vm.Config.MemoryAllocation != nil {
			info.ResourceAllocation.MemoryReservation = *vm.Config.MemoryAllocation.Reservation
			if vm.Config.MemoryAllocation.Limit != nil && *vm.Config.MemoryAllocation.Limit != -1 {
				info.ResourceAllocation.MemoryLimit = *vm.Config.MemoryAllocation.Limit
			}
			if vm.Config.MemoryAllocation.Shares != nil {
				info.ResourceAllocation.MemoryShares = vm.Config.MemoryAllocation.Shares.Shares
				info.ResourceAllocation.MemorySharesLevel = string(vm.Config.MemoryAllocation.Shares.Level)
			}
		}

		// Extract disk information
		info.Disks = s.extractDiskInfo(vm.Config.Hardware.Device)

		// Extract network adapter information
		info.NetworkAdapters = s.extractNetworkAdapters(vm.Config.Hardware.Device, vm.Guest)
	}

	// Runtime properties
	info.ConnectionState = string(vm.Runtime.ConnectionState)
	info.MaxCPUUsage = vm.Runtime.MaxCpuUsage
	info.MaxMemoryUsage = vm.Runtime.MaxMemoryUsage
	info.ConsolidationNeeded = vm.Runtime.ConsolidationNeeded != nil && *vm.Runtime.ConsolidationNeeded
	if vm.Runtime.FaultToleranceState != "" {
		info.FaultToleranceState = string(vm.Runtime.FaultToleranceState)
	}

	// Boot time and uptime
	if vm.Runtime.BootTime != nil {
		info.BootTime = *vm.Runtime.BootTime
		info.UptimeSeconds = int64(time.Since(*vm.Runtime.BootTime).Seconds())
	}

	// Host
	if vm.Runtime.Host != nil {
		info.Host = vm.Runtime.Host.Value
	}

	// Guest properties
	if vm.Guest != nil {
		info.ToolsStatus = string(vm.Guest.ToolsStatus)
		info.ToolsVersion = vm.Guest.ToolsVersion
		info.ToolsRunningStatus = vm.Guest.ToolsRunningStatus
		info.Hostname = vm.Guest.HostName
		info.GuestState = vm.Guest.GuestState

		// Collect all IP addresses from guest NICs
		var ipAddresses []string
		if vm.Guest.IpAddress != "" {
			ipAddresses = append(ipAddresses, vm.Guest.IpAddress)
		}
		for _, nic := range vm.Guest.Net {
			if nic.IpConfig != nil {
				for _, ipConfig := range nic.IpConfig.IpAddress {
					ip := ipConfig.IpAddress
					// Skip if already in list
					found := false
					for _, existing := range ipAddresses {
						if existing == ip {
							found = true
							break
						}
					}
					if !found && ip != "" {
						ipAddresses = append(ipAddresses, ip)
					}
				}
			}
		}
		info.IPAddresses = ipAddresses
	}

	// Guest heartbeat status
	info.GuestHeartbeatStatus = string(vm.GuestHeartbeatStatus)

	// Storage information from summary
	if vm.Summary.Storage.Committed > 0 {
		info.CommittedStorage = vm.Summary.Storage.Committed
	}
	if vm.Summary.Storage.Uncommitted > 0 {
		info.UncommittedStorage = vm.Summary.Storage.Uncommitted
	}

	// Datastores
	var datastores []string
	for _, ds := range vm.Datastore {
		datastores = append(datastores, ds.Value)
	}
	info.Datastores = datastores

	// Snapshot information
	if vm.Snapshot != nil {
		info.Snapshots = s.extractSnapshotInfo(vm.Snapshot.RootSnapshotList)
		if vm.Snapshot.CurrentSnapshot != nil {
			info.CurrentSnapshot = vm.Snapshot.CurrentSnapshot.Value
		}
	}

	// File layout
	if vm.LayoutEx != nil {
		var configFiles []string
		var logFiles []string
		for _, file := range vm.LayoutEx.File {
			if strings.HasSuffix(file.Name, ".vmx") || strings.HasSuffix(file.Name, ".nvram") {
				configFiles = append(configFiles, file.Name)
			} else if strings.HasSuffix(file.Name, ".log") {
				logFiles = append(logFiles, file.Name)
			}
		}
		info.ConfigFiles = configFiles
		info.LogFiles = logFiles
	}

	// Resource pool
	if vm.ResourcePool != nil {
		info.ResourcePool = vm.ResourcePool.Value
	}

	// Parent (folder)
	if vm.Parent != nil {
		info.Folder = vm.Parent.Value
	}

	return info
}

// matchesFilter checks if a VM matches the given filter criteria
func (s *Service) matchesFilter(vm VMInfo, filter VMFilter) bool {
	if filter.PowerState != "" && !strings.EqualFold(vm.PowerState, filter.PowerState) {
		return false
	}

	if filter.Name != "" && !strings.Contains(strings.ToLower(vm.Name), strings.ToLower(filter.Name)) {
		return false
	}

	// GuestOS filtering not supported with minimal properties
	// Cluster filtering not supported with minimal properties

	return true
}
