package vmhandler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nirarg/vm-deep-inspection-demo/pkg/types"
	"github.com/sirupsen/logrus"
)

// CreateVMSnapshot godoc
// @Summary Create a VM snapshot
// @Description Create a snapshot for a specific virtual machine
// @Tags vms
// @Accept json
// @Produce json
// @Param name path string true "VM name" example("web-server-01")
// @Param request body types.SnapshotCreateRequest true "Snapshot creation request"
// @Success 200 {object} types.SnapshotCreateResponse "Snapshot created successfully"
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 404 {object} types.ErrorResponse "VM not found"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Failure 503 {object} types.ErrorResponse "vSphere connection unavailable"
// @Router /api/v1/vms/{name}/snapshot [post]
func (h *VMHandler) CreateVMSnapshot(c *gin.Context) {
	// Get VM name from query parameter
	vmName := c.Param("name")
	if vmName == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "VM name is required",
			Code:    "MISSING_VM_NAME",
			Details: "Please provide VM name as query parameter: ?name=xxx",
		})
		return
	}

	// Parse request body
	var req types.SnapshotCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind snapshot request")
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"vm_name":       vmName,
		"snapshot_name": req.Name,
		"memory":        req.Memory,
		"quiesce":       req.Quiesce,
	}).Info("Creating VM snapshot")

	// Create snapshot
	snapshotID, err := h.vmService.CreateSnapshot(
		c.Request.Context(),
		vmName,
		req.Name,
		req.Description,
		req.Memory,
		req.Quiesce,
	)

	if err != nil {
		h.logger.WithError(err).Error("Failed to create snapshot")

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
			Error:   "Failed to create snapshot",
			Code:    "SNAPSHOT_CREATE_FAILED",
			Details: err.Error(),
		})
		return
	}

	response := types.SnapshotCreateResponse{
		SnapshotID: snapshotID,
		Name:       req.Name,
		VMID:       "",
		VMName:     vmName,
		Status:     "completed",
		Message:    "Snapshot created successfully",
	}

	h.logger.WithFields(logrus.Fields{
		"snapshot_id": snapshotID,
		"vm_name":     vmName,
	}).Info("Snapshot created successfully")

	c.JSON(http.StatusOK, response)
}

// RemoveVMSnapshot godoc
// @Summary Delete a VM snapshot
// @Description Delete a snapshot from a specific virtual machine
// @Tags vms
// @Accept json
// @Produce json
// @Param name path string true "VM name" example("web-server-01")
// @Param snapshotName path string true "Snapshot name" example("snapshot-123")
// @Param consolidate query bool false "Whether to consolidate the snapshot (default: false)"
// @Success 200 {object} types.SnapshotDeleteResponse "Snapshot deleted successfully"
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 404 {object} types.ErrorResponse "VM or snapshot not found"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Failure 503 {object} types.ErrorResponse "vSphere connection unavailable"
// @Router /api/v1/vms/{name}/snapshot/{snapshotName} [delete]
func (h *VMHandler) RemoveVMSnapshot(c *gin.Context) {
	// Get VM name
	vmName := c.Param("name")
	if vmName == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "VM name is required",
			Code:    "MISSING_VM_NAME",
			Details: "Please provide VM name as query parameter: ?name=xxx",
		})
		return
	}

	// Get snapshot ID
	snapshotName := c.Param("snapshotName")
	if snapshotName == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Snapshot ID is required",
			Code:    "MISSING_SNAPSHOT_ID",
			Details: "Please provide snapshot ID in path parameter",
		})
		return
	}

	// Use consolidate = false by default. means - don't merge the snapshot to disk when deleting it.
	consolidateStr := c.DefaultQuery("consolidate", "false")
	consolidate, err := strconv.ParseBool(consolidateStr)

	h.logger.WithFields(logrus.Fields{
		"vm_name":       vmName,
		"snapshot_name": snapshotName,
	}).Info("Deleting VM snapshot")

	// Delete snapshot
	if _, err = h.vmService.RemoveSnapshot(
		c.Request.Context(),
		vmName,
		snapshotName,
		&consolidate,
	); err != nil {
		h.logger.WithError(err).Error("Failed to delete snapshot")

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
				Error:   "VM or snapshot not found",
				Code:    "SNAPSHOT_NOT_FOUND",
				Details: err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Failed to delete snapshot",
			Code:    "SNAPSHOT_DELETE_FAILED",
			Details: err.Error(),
		})
		return
	}

	response := types.SnapshotDeleteResponse{
		SnapshotName: snapshotName,
		VMName:       vmName,
		Status:       "completed",
		Message:      "Snapshot deleted successfully",
	}

	h.logger.WithFields(logrus.Fields{
		"vm_name":       vmName,
		"snapshot_name": snapshotName,
	}).Info("Snapshot deleted successfully")

	c.JSON(http.StatusOK, response)
}
