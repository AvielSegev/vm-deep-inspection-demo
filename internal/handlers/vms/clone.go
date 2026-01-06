package vmhandler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nirarg/vm-deep-inspection-demo/pkg/types"
	"github.com/sirupsen/logrus"
)

// CreateClone godoc
// @Summary Create a clone from VM snapshot
// @Description Create a linked clone from a VM snapshot for inspection
// @Tags vms
// @Accept json
// @Produce json
// @Param name query string true "VM name" example("web-server-01")
// @Param request body types.CloneRequest true "Clone request"
// @Success 200 {object} types.CloneResponse "Clone created successfully"
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 404 {object} types.ErrorResponse "VM or snapshot not found"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/vms/clone [post]
func (h *VMHandler) CreateClone(c *gin.Context) {
	vmName := c.Query("name")
	if vmName == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "VM name is required",
			Code:    "MISSING_VM_NAME",
			Details: "Please provide VM name as query parameter: ?name=xxx",
		})
		return
	}

	var req types.CloneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind clone request")
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Invalid request body",
			Code:    "INVALID_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Generate clone name if not provided
	cloneName := req.CloneName
	if cloneName == "" {
		cloneName = vmName + "-clone-" + time.Now().Format("20060102150405")
	}

	h.logger.WithFields(logrus.Fields{
		"vm_name":       vmName,
		"snapshot_name": req.SnapshotName,
		"clone_name":    cloneName,
	}).Info("Creating clone from snapshot")

	// Find snapshot
	snapshotRef, err := h.vmService.FindSnapshotByName(c.Request.Context(), vmName, req.SnapshotName)
	if err != nil {
		h.logger.WithError(err).Error("Failed to find snapshot")
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "Snapshot not found",
				Code:    "SNAPSHOT_NOT_FOUND",
				Details: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Failed to find snapshot",
			Code:    "SNAPSHOT_FIND_FAILED",
			Details: err.Error(),
		})
		return
	}

	// Create clone
	err = h.vmService.CreateLinkedClone(c.Request.Context(), vmName, snapshotRef, cloneName)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create clone")
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Failed to create clone",
			Code:    "CLONE_CREATE_FAILED",
			Details: err.Error(),
		})
		return
	}

	response := types.CloneResponse{
		CloneName:    cloneName,
		VMName:       vmName,
		SnapshotName: req.SnapshotName,
		Status:       "completed",
		Message:      "Clone created successfully",
	}

	h.logger.WithFields(logrus.Fields{
		"clone_name": cloneName,
	}).Info("Clone created successfully")

	c.JSON(http.StatusOK, response)
}

// DeleteClone godoc
// @Summary Delete a cloned VM
// @Description Delete a cloned VM created for inspection
// @Tags vms
// @Accept json
// @Produce json
// @Param name query string true "Clone VM name" example("web-server-01-clone-123")
// @Success 200 {object} types.ErrorResponse "Clone deleted successfully"
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 404 {object} types.ErrorResponse "Clone not found"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/vms/delete-clone [delete]
func (h *VMHandler) DeleteClone(c *gin.Context) {
	cloneName := c.Query("name")
	if cloneName == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Clone name is required",
			Code:    "MISSING_CLONE_NAME",
			Details: "Please provide clone name as query parameter: ?name=xxx",
		})
		return
	}

	h.logger.WithField("clone_name", cloneName).Info("Deleting clone")

	err := h.vmService.DeleteVM(c.Request.Context(), cloneName)
	if err != nil {
		h.logger.WithError(err).Error("Failed to delete clone")
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, types.ErrorResponse{
				Error:   "Clone not found",
				Code:    "CLONE_NOT_FOUND",
				Details: err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Failed to delete clone",
			Code:    "CLONE_DELETE_FAILED",
			Details: err.Error(),
		})
		return
	}

	h.logger.Info("Clone deleted successfully")
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Clone deleted successfully",
	})
}
