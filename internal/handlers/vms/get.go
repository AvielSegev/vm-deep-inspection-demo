package vmhandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nirarg/vm-deep-inspection-demo/pkg/types"
	"github.com/sirupsen/logrus"
)

// GetVM godoc
// @Summary Get virtual machine details
// @Description Get detailed information about a specific virtual machine by name
// @Tags vms
// @Accept json
// @Produce json
// @Param name path string true "VM name" example("web-server-01")
// @Success 200 {object} types.VMDetailsResponse "Virtual machine details"
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 404 {object} types.ErrorResponse "VM not found"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Failure 503 {object} types.ErrorResponse "vSphere connection unavailable"
// @Router /api/v1/vms/{name} [get]
func (h *VMHandler) GetVM(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "VM name is required",
			Code:    "MISSING_VM_NAME",
			Details: "VM name must be provided in the URL path",
		})
		return
	}

	h.logger.WithField("vm_name", name).Info("Getting VM details")

	result, err := h.vmService.GetVMByName(c.Request.Context(), name)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get VM")

		if isConnectionError(err) {
			c.JSON(http.StatusServiceUnavailable, types.ErrorResponse{
				Error:   "vSphere connection unavailable",
				Code:    "VSPHERE_UNAVAILABLE",
				Details: "Unable to connect to vSphere. Please try again later.",
			})
			return
		}

		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "VM not found",
				Code:    "VM_NOT_FOUND",
				Details: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Failed to retrieve VM",
			Code:    "VM_GET_FAILED",
			Details: "An error occurred while retrieving the virtual machine",
		})
		return
	}

	// Convert detailed VM info to API response
	vmBasic := types.VM{
		UUID:       result.VM.UUID,
		Name:       result.VM.Name,
		PowerState: result.VM.PowerState,
	}

	// Convert disks
	var disks []types.VMDisk
	for _, disk := range result.VM.Disks {
		disks = append(disks, types.VMDisk{
			Label:           disk.Label,
			CapacityKB:      disk.CapacityKB,
			CapacityGB:      disk.CapacityKB / 1024 / 1024,
			DiskPath:        disk.DiskPath,
			Datastore:       disk.Datastore,
			ThinProvisioned: disk.ThinProvisioned,
			DiskMode:        disk.DiskMode,
		})
	}

	// Convert network adapters
	var networkAdapters []types.VMNetworkAdapter
	for _, adapter := range result.VM.NetworkAdapters {
		networkAdapters = append(networkAdapters, types.VMNetworkAdapter{
			Label:       adapter.Label,
			NetworkName: adapter.NetworkName,
			MacAddress:  adapter.MacAddress,
			IPAddresses: adapter.IPAddresses,
			Connected:   adapter.Connected,
			AdapterType: adapter.AdapterType,
		})
	}

	// Convert snapshots
	var snapshots []types.VMSnapshot
	for _, snap := range result.VM.Snapshots {
		snapshots = append(snapshots, types.VMSnapshot{
			Name:        snap.Name,
			Description: snap.Description,
			CreateTime:  snap.CreateTime,
			State:       snap.State,
			Quiesced:    snap.Quiesced,
			ID:          snap.ID,
		})
	}

	// Build detailed response with all available information
	response := types.VMDetailsResponse{
		VM: vmBasic,
		Hardware: types.VMHardwareInfo{
			NumCPU:            result.VM.NumCPU,
			NumCoresPerSocket: result.VM.NumCoresPerSocket,
			MemoryMB:          result.VM.MemoryMB,
			GuestFullName:     result.VM.GuestFullName,
			Version:           result.VM.Version,
			FirmwareType:      result.VM.FirmwareType,
		},
		Tools: types.VMToolsInfo{
			Status:        result.VM.ToolsStatus,
			Version:       result.VM.ToolsVersion,
			RunningStatus: result.VM.ToolsRunningStatus,
		},
		GuestInfo: types.VMGuestInfo{
			Hostname:             result.VM.Hostname,
			IPAddresses:          result.VM.IPAddresses,
			GuestID:              result.VM.GuestID,
			GuestState:           result.VM.GuestState,
			GuestHeartbeatStatus: result.VM.GuestHeartbeatStatus,
		},
		Metadata: types.VMMetadata{
			InstanceUUID: result.VM.InstanceUUID,
			BiosUUID:     result.VM.BiosUUID,
			Annotation:   result.VM.Annotation,
			Template:     result.VM.Template,
		},
		Runtime: types.VMRuntimeInfo{
			Host:                result.VM.Host,
			ConnectionState:     result.VM.ConnectionState,
			BootTime:            result.VM.BootTime,
			UptimeSeconds:       result.VM.UptimeSeconds,
			MaxCPUUsage:         result.VM.MaxCPUUsage,
			MaxMemoryUsage:      result.VM.MaxMemoryUsage,
			ConsolidationNeeded: result.VM.ConsolidationNeeded,
			FaultToleranceState: result.VM.FaultToleranceState,
		},
		Disks:           disks,
		NetworkAdapters: networkAdapters,
		Snapshots:       snapshots,
		CurrentSnapshot: result.VM.CurrentSnapshot,
		Resources: types.VMResourceInfo{
			CPUReservationMHz:   result.VM.ResourceAllocation.CPUReservation,
			CPULimitMHz:         result.VM.ResourceAllocation.CPULimit,
			CPUShares:           result.VM.ResourceAllocation.CPUShares,
			CPUSharesLevel:      result.VM.ResourceAllocation.CPUSharesLevel,
			MemoryReservationMB: result.VM.ResourceAllocation.MemoryReservation,
			MemoryLimitMB:       result.VM.ResourceAllocation.MemoryLimit,
			MemoryShares:        result.VM.ResourceAllocation.MemoryShares,
			MemorySharesLevel:   result.VM.ResourceAllocation.MemorySharesLevel,
		},
		Storage: types.VMStorageSummary{
			CommittedBytes:   result.VM.CommittedStorage,
			CommittedGB:      result.VM.CommittedStorage / 1024 / 1024 / 1024,
			UncommittedBytes: result.VM.UncommittedStorage,
			UncommittedGB:    result.VM.UncommittedStorage / 1024 / 1024 / 1024,
			Datastores:       result.VM.Datastores,
		},
		Files: types.VMFileInfo{
			VMPathName:  result.VM.VMPathName,
			ConfigFiles: result.VM.ConfigFiles,
			LogFiles:    result.VM.LogFiles,
		},
		Location: types.VMLocationInfo{
			Folder:       result.VM.Folder,
			ResourcePool: result.VM.ResourcePool,
		},
		Advanced: types.VMAdvancedInfo{
			CPUHotAddEnabled:      result.VM.CPUHotAddEnabled,
			CPUHotRemoveEnabled:   result.VM.CPUHotRemoveEnabled,
			MemoryHotAddEnabled:   result.VM.MemoryHotAddEnabled,
			ChangeTrackingEnabled: result.VM.ChangeTrackingEnabled,
		},
	}

	h.logger.WithFields(logrus.Fields{
		"vm_name": vmBasic.Name,
		"vm_uuid": vmBasic.UUID,
	}).Info("Successfully retrieved VM details")

	c.JSON(http.StatusOK, response)
}
