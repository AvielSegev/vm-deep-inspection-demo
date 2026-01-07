package vmhandler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubev2v/vm-migration-detective/pkg/checks"
	"github.com/nirarg/vm-deep-inspection-demo/pkg/types"
	"github.com/sirupsen/logrus"
)

// RunCheck godoc
// @Summary Run validation checks on a VM snapshot
// @Description Run validation checks on a VM snapshot. If check parameter is provided, runs that specific check. If omitted, runs all available checks.
// @Tags vms
// @Accept json
// @Produce json
// @Param vms query string true "Original VM name" example("web-server-01")
// @Param snapshot query string true "Snapshot name" example("inspection-snapshot")
// @Param check query string false "Check type to run (fstab, disk-access). If omitted, runs all checks." example("fstab")
// @Success 200 {object} types.CheckResponse "Check completed successfully"
// @Failure 400 {object} types.ErrorResponse "Invalid request"
// @Failure 404 {object} types.ErrorResponse "VM or snapshot not found"
// @Failure 500 {object} types.ErrorResponse "Internal server error"
// @Router /api/v1/vms/check [post]
func (h *VMHandler) RunCheck(c *gin.Context) {
	vmName := c.Query("vms")
	snapshotName := c.Query("snapshot")
	checkType := c.Query("check")

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

	logFields := logrus.Fields{
		"vm_name":       vmName,
		"snapshot_name": snapshotName,
	}
	if checkType != "" {
		logFields["check_type"] = checkType
		h.logger.WithFields(logFields).Info("Running specific validation check on VM snapshot")
	} else {
		h.logger.WithFields(logFields).Info("Running all validation checks on VM snapshot")
	}

	// Get datacenter name
	datacenter, err := h.vmService.GetDatacenterName(c.Request.Context(), vmName)
	if err != nil {
		h.logger.WithError(err).Error("failed to get datacenter name")
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Check failed",
			Code:    "CHECK_FAILED",
			Details: err.Error(),
		})
		return
	}

	// Get snapshot disk info
	h.logger.Debug("Getting snapshot disk info from vm_service")
	diskInfo, err := h.vmService.GetSnapshotDiskInfo(c.Request.Context(), vmName, snapshotName)
	if err != nil {
		h.logger.WithError(err).Error("failed to get snapshot disk info")
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Check failed",
			Code:    "CHECK_FAILED",
			Details: fmt.Sprintf("failed to get snapshot disk info: %v", err),
		})
		return
	}

	// Get vCenter credentials from vmClient
	vcenterURL := h.vmService.GetVCenterURL()
	username, password := h.vmService.GetCredentials()

	// Create inspection params
	params := checks.InspectionParams{
		Ctx:          c.Request.Context(),
		VMName:       vmName,
		SnapshotName: snapshotName,
		Datacenter:   datacenter,
		VCenterURL:   vcenterURL,
		Username:     username,
		Password:     password,
		DiskInfo:     diskInfo,
		DB:           h.inspector.GetDB(),
		Logger:       h.logger,
	}

	// Define all available checks
	allChecks := map[string]checks.Check{
		"fstab":       checks.NewFstabCheck(),
		"disk-access": checks.NewDiskAccessCheck(),
	}

	// Determine which checks to run
	var checksToRun map[string]checks.Check
	if checkType != "" {
		// Run specific check
		check, exists := allChecks[checkType]
		if !exists {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{
				Error:   "Unknown check type",
				Code:    "UNKNOWN_CHECK_TYPE",
				Details: fmt.Sprintf("check type '%s' is not supported. Supported types: fstab, disk-access", checkType),
			})
			return
		}
		checksToRun = map[string]checks.Check{checkType: check}
	} else {
		// Run all checks
		checksToRun = allChecks
	}

	// Execute all selected checks
	var results []types.CheckResult
	allValid := true

	for name, check := range checksToRun {
		h.logger.WithField("check_type", name).Info("Executing validation check")
		result := check.Run(params)

		results = append(results, types.CheckResult{
			CheckType: name,
			Valid:     result.Valid,
			Message:   result.Message,
			Error:     result.Error,
		})

		if !result.Valid {
			allValid = false
		}

		h.logger.WithFields(logrus.Fields{
			"check_type": name,
			"valid":      result.Valid,
		}).Info("Validation check completed")
	}

	response := types.CheckResponse{
		VMName:       vmName,
		SnapshotName: snapshotName,
		Results:      results,
		AllValid:     allValid,
	}

	h.logger.WithFields(logrus.Fields{
		"checks_run": len(results),
		"all_valid":  allValid,
	}).Info("All validation checks completed")

	c.JSON(http.StatusOK, response)
}
