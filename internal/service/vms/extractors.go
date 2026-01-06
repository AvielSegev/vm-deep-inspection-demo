package vms

import (
	"strings"

	vimtypes "github.com/vmware/govmomi/vim25/types"
)

// extractDiskInfo extracts disk information from hardware devices
func (s *Service) extractDiskInfo(devices []vimtypes.BaseVirtualDevice) []VMDiskInfo {
	var disks []VMDiskInfo
	for _, device := range devices {
		if disk, ok := device.(*vimtypes.VirtualDisk); ok {
			diskInfo := VMDiskInfo{
				Label:         disk.DeviceInfo.GetDescription().Label,
				CapacityKB:    disk.CapacityInKB,
				ControllerKey: disk.ControllerKey,
			}

			if backing, ok := disk.Backing.(*vimtypes.VirtualDiskFlatVer2BackingInfo); ok {
				diskInfo.DiskPath = backing.FileName
				diskInfo.ThinProvisioned = *backing.ThinProvisioned
				diskInfo.DiskMode = backing.DiskMode
				// Extract datastore from path [datastore] path/to/disk.vmdk
				if idx := strings.Index(backing.FileName, "]"); idx > 0 {
					diskInfo.Datastore = backing.FileName[1:idx]
				}
			}

			disks = append(disks, diskInfo)
		}
	}
	return disks
}

// extractNetworkAdapters extracts network adapter information from hardware devices
func (s *Service) extractNetworkAdapters(devices []vimtypes.BaseVirtualDevice, guest *vimtypes.GuestInfo) []VMNetworkAdapterInfo {
	var adapters []VMNetworkAdapterInfo

	// Create a map of MAC to IPs from guest info
	macToIPs := make(map[string][]string)
	if guest != nil {
		for _, nic := range guest.Net {
			if nic.MacAddress != "" && nic.IpConfig != nil {
				var ips []string
				for _, ipConfig := range nic.IpConfig.IpAddress {
					if ipConfig.IpAddress != "" {
						ips = append(ips, ipConfig.IpAddress)
					}
				}
				macToIPs[nic.MacAddress] = ips
			}
		}
	}

	for _, device := range devices {
		var label, mac, network, adapterType string
		var connected bool

		switch nic := device.(type) {
		case *vimtypes.VirtualE1000:
			label = nic.DeviceInfo.GetDescription().Label
			mac = nic.MacAddress
			connected = nic.Connectable != nil && nic.Connectable.Connected
			adapterType = "E1000"
			if backing, ok := nic.Backing.(*vimtypes.VirtualEthernetCardNetworkBackingInfo); ok {
				network = backing.DeviceName
			}
		case *vimtypes.VirtualE1000e:
			label = nic.DeviceInfo.GetDescription().Label
			mac = nic.MacAddress
			connected = nic.Connectable != nil && nic.Connectable.Connected
			adapterType = "E1000e"
			if backing, ok := nic.Backing.(*vimtypes.VirtualEthernetCardNetworkBackingInfo); ok {
				network = backing.DeviceName
			}
		case *vimtypes.VirtualVmxnet3:
			label = nic.DeviceInfo.GetDescription().Label
			mac = nic.MacAddress
			connected = nic.Connectable != nil && nic.Connectable.Connected
			adapterType = "VMXNET3"
			if backing, ok := nic.Backing.(*vimtypes.VirtualEthernetCardNetworkBackingInfo); ok {
				network = backing.DeviceName
			}
		default:
			continue
		}

		adapter := VMNetworkAdapterInfo{
			Label:       label,
			NetworkName: network,
			MacAddress:  mac,
			Connected:   connected,
			AdapterType: adapterType,
			IPAddresses: macToIPs[mac],
		}
		adapters = append(adapters, adapter)
	}

	return adapters
}

// extractSnapshotInfo recursively extracts snapshot information
func (s *Service) extractSnapshotInfo(snapshots []vimtypes.VirtualMachineSnapshotTree) []VMSnapshotInfo {
	var result []VMSnapshotInfo
	for _, snap := range snapshots {
		info := VMSnapshotInfo{
			Name:        snap.Name,
			Description: snap.Description,
			CreateTime:  snap.CreateTime,
			State:       string(snap.State),
			Quiesced:    snap.Quiesced,
			ID:          snap.Id,
		}
		result = append(result, info)

		// Recursively add child snapshots
		if len(snap.ChildSnapshotList) > 0 {
			result = append(result, s.extractSnapshotInfo(snap.ChildSnapshotList)...)
		}
	}
	return result
}
