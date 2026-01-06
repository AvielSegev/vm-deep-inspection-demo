package vmhandler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nirarg/vm-deep-inspection-demo/pkg/types"
	"github.com/sirupsen/logrus"
)

// InspectSnapshot godoc
// @Summary Inspect a VM snapshot directly
// @Description Run virt-inspector or virt-v2v-inspector on a VM snapshot using VDDK
// @Tags vms
// @Accept json
// @Produce json
// @Param vms query string true "Original VM name" example("web-server-01")
// @Param snapshot query string true "Snapshot name" example("inspection-snapshot")
// @Param inspector query string false "Inspector type: 'virt-inspector' (default) or 'virt-v2v-inspector'" example("virt-inspector")
// @Success 200 {object} types.VMInspectionResponse "Inspection completed successfully"
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 404 {object} types.ErrorResponse "VM or snapshot not found"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/vms/inspect-snapshot [post]
func (h *VMHandler) InspectSnapshot(c *gin.Context) {
	vmName := c.Query("vms")
	snapshotName := c.Query("snapshot")
	inspectorType := c.DefaultQuery("inspector", "virt-inspector") // Default to virt-inspector

	if vmName == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "VM name is required",
			Code:    "MISSING_VM_NAME",
			Details: "Please provide VM name as query parameter: ?vms=xxx",
		})
		return
	}

	if snapshotName == "" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Snapshot name is required",
			Code:    "MISSING_SNAPSHOT_NAME",
			Details: "Please provide snapshot name as query parameter: &snapshot=xxx",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"vm_name":        vmName,
		"snapshot_name":  snapshotName,
		"inspector_type": inspectorType,
	}).Info("Inspecting VM snapshot with VDDK")

	// Validate inspector type
	if inspectorType != "virt-inspector" && inspectorType != "virt-v2v-inspector" {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Invalid inspector type",
			Code:    "INVALID_INSPECTOR_TYPE",
			Details: fmt.Sprintf("inspector must be 'virt-inspector' or 'virt-v2v-inspector', got: %s", inspectorType),
		})
		return
	}

	// SSL verification option for vpx:// URL
	// Using no_verify=1 for now to simplify (can be enhanced later with certificate support)
	sslVerify := "no_verify=1"

	datacenter, err := h.vmService.GetDatacenterName(c.Request.Context(), vmName)
	if err != nil {
		h.logger.WithError(err).Error("failed to get datacenter name")
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Inspection failed",
			Code:    "INSPECTION_FAILED",
			Details: err.Error(),
		})
		return
	}

	// Get snapshot disk info (morefs and disk path) from vm_service
	h.logger.Debug("Getting snapshot disk info from vm_service")
	diskInfo, err := h.vmService.GetSnapshotDiskInfo(c.Request.Context(), vmName, snapshotName)
	if err != nil {
		h.logger.WithError(err).Error("failed to get snapshot disk info")
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Inspection failed",
			Code:    "INSPECTION_FAILED",
			Details: fmt.Sprintf("failed to get snapshot disk info: %v", err),
		})
		return
	}

	// Use the selected inspector to inspect snapshot
	var response types.VMInspectionResponse
	message := fmt.Sprintf("Snapshot inspection completed successfully using %s", inspectorType)

	if inspectorType == "virt-v2v-inspector" {
		h.logger.Info("Running virt-v2v-inspector with VDDK on snapshot")
		inspectionData, err := h.inspector.InspectWithVirtV2v(
			c.Request.Context(),
			vmName,
			snapshotName,
			datacenter,
			diskInfo,
			sslVerify,
		)
		if err != nil {
			h.logger.WithError(err).WithField("inspector_type", inspectorType).Error("inspection execution failed")
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "Inspection failed",
				Code:    "INSPECTION_FAILED",
				Details: err.Error(),
			})
			return
		}
		response = types.NewVirtV2VInspectorResponse(vmName, snapshotName, message, inspectionData)
	} else {
		// Default: use virt-inspector
		h.logger.Info("Running virt-inspector with VDDK on snapshot")
		inspectionData, err := h.inspector.InspectWithVirt(
			c.Request.Context(),
			vmName,
			snapshotName,
			datacenter,
			diskInfo,
		)
		if err != nil {
			h.logger.WithError(err).WithField("inspector_type", inspectorType).Error("inspection execution failed")
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{
				Error:   "Inspection failed",
				Code:    "INSPECTION_FAILED",
				Details: err.Error(),
			})
			return
		}
		response = types.NewVirtInspectorResponse(vmName, snapshotName, message, inspectionData)
	}

	h.logger.WithField("inspector_type", inspectorType).Info("Snapshot inspection completed successfully")
	c.JSON(http.StatusOK, response)
}
