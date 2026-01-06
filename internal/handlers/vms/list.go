package vmhandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nirarg/vm-deep-inspection-demo/internal/service/vms"
	"github.com/nirarg/vm-deep-inspection-demo/pkg/types"
)

// ListVMs godoc
// @Summary List all virtual machines
// @Description Get a list of all virtual machines with optional name filtering
// @Tags vms
// @Accept json
// @Produce json
// @Param name_contains query string false "Filter VMs where name contains this string" example("web")
// @Success 200 {object} types.VMListResponse "List of virtual machines"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Failure 503 {object} types.ErrorResponse "vSphere connection unavailable"
// @Router /api/v1/vms [get]
func (h *VMHandler) ListVMs(c *gin.Context) {
	nameContains := c.Query("name_contains")

	h.logger.WithField("name_contains", nameContains).Info("Listing VMs")

	// Build filter from query parameters
	filter := vms.VMFilter{
		Name: nameContains,
	}

	result, err := h.vmService.ListVMs(c.Request.Context(), filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list VMs")

		if isConnectionError(err) {
			c.JSON(http.StatusServiceUnavailable, types.ErrorResponse{
				Error:   "vSphere connection unavailable",
				Code:    "VSPHERE_UNAVAILABLE",
				Details: "Unable to connect to vSphere. Please try again later.",
			})
			return
		}

		if isAuthenticationError(err) {
			c.JSON(http.StatusServiceUnavailable, types.ErrorResponse{
				Error:   "vSphere authentication failed",
				Code:    "VSPHERE_AUTH_FAILED",
				Details: "Authentication to vSphere failed. Please check configuration.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Failed to retrieve VMs",
			Code:    "VM_LIST_FAILED",
			Details: "An error occurred while retrieving virtual machines from vSphere",
		})
		return
	}

	// Convert VMInfos to VMs
	var vms []types.VM
	for _, vmInfo := range result.VMs {
		vms = append(vms, h.convertVMInfoToVM(vmInfo))
	}

	response := types.VMListResponse{
		Datacenter: result.Datacenter,
		VMs:        vms,
		Total:      result.Total,
	}

	h.logger.WithField("total_vms", result.Total).Info("Successfully retrieved VMs")

	c.JSON(http.StatusOK, response)
}
