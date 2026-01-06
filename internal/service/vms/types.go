package vms

import "time"

// VMFilter contains filtering options for VM discovery
type VMFilter struct {
	Datacenter string `json:"datacenter,omitempty"`
	Cluster    string `json:"cluster,omitempty"`
	PowerState string `json:"power_state,omitempty"`
	Name       string `json:"name,omitempty"`
	GuestOS    string `json:"guest_os,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

// VMInfo represents basic information about a virtual machine
type VMInfo struct {
	UUID       string `json:"uuid"`
	Name       string `json:"name"`
	PowerState string `json:"power_state"`
}

// VMDiskInfo represents virtual disk information
type VMDiskInfo struct {
	Label           string `json:"label"`
	CapacityKB      int64  `json:"capacity_kb"`
	DiskPath        string `json:"disk_path"`
	Datastore       string `json:"datastore"`
	ThinProvisioned bool   `json:"thin_provisioned"`
	DiskMode        string `json:"disk_mode"`
	ControllerKey   int32  `json:"controller_key"`
}

// VMNetworkAdapterInfo represents network adapter information
type VMNetworkAdapterInfo struct {
	Label       string   `json:"label"`
	NetworkName string   `json:"network_name"`
	MacAddress  string   `json:"mac_address"`
	IPAddresses []string `json:"ip_addresses"`
	Connected   bool     `json:"connected"`
	AdapterType string   `json:"adapter_type"`
}

// VMSnapshotInfo represents snapshot information
type VMSnapshotInfo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreateTime  time.Time `json:"create_time"`
	State       string    `json:"state"`
	Quiesced    bool      `json:"quiesced"`
	ID          int32     `json:"id"`
}

// VMResourceAllocation represents resource allocation settings
type VMResourceAllocation struct {
	CPUReservation    int64  `json:"cpu_reservation_mhz"`
	CPULimit          int64  `json:"cpu_limit_mhz"`
	CPUShares         int32  `json:"cpu_shares"`
	CPUSharesLevel    string `json:"cpu_shares_level"`
	MemoryReservation int64  `json:"memory_reservation_mb"`
	MemoryLimit       int64  `json:"memory_limit_mb"`
	MemoryShares      int32  `json:"memory_shares"`
	MemorySharesLevel string `json:"memory_shares_level"`
}

// VMDetailedInfo represents comprehensive information about a virtual machine
type VMDetailedInfo struct {
	// Basic Info
	UUID          string `json:"uuid"`
	Name          string `json:"name"`
	PowerState    string `json:"power_state"`
	GuestFullName string `json:"guest_full_name"`
	GuestID       string `json:"guest_id"`
	InstanceUUID  string `json:"instance_uuid"`
	BiosUUID      string `json:"bios_uuid"`
	Annotation    string `json:"annotation"`

	// Hardware
	NumCPU              int32  `json:"num_cpu"`
	NumCoresPerSocket   int32  `json:"num_cores_per_socket"`
	MemoryMB            int32  `json:"memory_mb"`
	Version             string `json:"version"`
	FirmwareType        string `json:"firmware_type"`
	CPUHotAddEnabled    bool   `json:"cpu_hot_add_enabled"`
	CPUHotRemoveEnabled bool   `json:"cpu_hot_remove_enabled"`
	MemoryHotAddEnabled bool   `json:"memory_hot_add_enabled"`

	// Guest Info
	ToolsStatus        string   `json:"tools_status"`
	ToolsVersion       string   `json:"tools_version"`
	ToolsRunningStatus string   `json:"tools_running_status"`
	IPAddresses        []string `json:"ip_addresses"`
	Hostname           string   `json:"hostname"`
	GuestState         string   `json:"guest_state"`

	// Runtime Info
	Host                string    `json:"host"`
	ConnectionState     string    `json:"connection_state"`
	BootTime            time.Time `json:"boot_time,omitempty"`
	UptimeSeconds       int64     `json:"uptime_seconds"`
	MaxCPUUsage         int32     `json:"max_cpu_usage_mhz"`
	MaxMemoryUsage      int32     `json:"max_memory_usage_mb"`
	ConsolidationNeeded bool      `json:"consolidation_needed"`

	// Storage
	Disks              []VMDiskInfo `json:"disks"`
	Datastores         []string     `json:"datastores"`
	CommittedStorage   int64        `json:"committed_storage_bytes"`
	UncommittedStorage int64        `json:"uncommitted_storage_bytes"`

	// Network
	NetworkAdapters []VMNetworkAdapterInfo `json:"network_adapters"`

	// Resource Allocation
	ResourceAllocation VMResourceAllocation `json:"resource_allocation"`

	// Location
	Folder       string `json:"folder"`
	ResourcePool string `json:"resource_pool"`

	// Snapshots
	Snapshots       []VMSnapshotInfo `json:"snapshots"`
	CurrentSnapshot string           `json:"current_snapshot"`

	// Files
	VMPathName  string   `json:"vm_path_name"`
	ConfigFiles []string `json:"config_files"`
	LogFiles    []string `json:"log_files"`

	// Advanced
	Template              bool   `json:"template"`
	ChangeTrackingEnabled bool   `json:"change_tracking_enabled"`
	FaultToleranceState   string `json:"fault_tolerance_state"`
	GuestHeartbeatStatus  string `json:"guest_heartbeat_status"`
}

// VMResult represents a single VM result
type VMResult struct {
	Datacenter string `json:"datacenter"`
	VM         VMInfo `json:"vm"`
}

// VMDetailedResult represents a detailed VM result
type VMDetailedResult struct {
	Datacenter string         `json:"datacenter"`
	VM         VMDetailedInfo `json:"vm"`
}

// VMListResult represents the result of VM listing
type VMListResult struct {
	Datacenter string   `json:"datacenter"`
	VMs        []VMInfo `json:"vms"`
	Total      int      `json:"total"`
}
