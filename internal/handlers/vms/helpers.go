package vmhandler

import (
	"github.com/nirarg/vm-deep-inspection-demo/internal/service/vms"
	"github.com/nirarg/vm-deep-inspection-demo/pkg/types"
)

// convertVMInfoToVM converts internal VMInfo to API VM type
func (h *VMHandler) convertVMInfoToVM(vmInfo vms.VMInfo) types.VM {
	return types.VM{
		UUID:       vmInfo.UUID,
		Name:       vmInfo.Name,
		PowerState: vmInfo.PowerState,
	}
}

// Helper functions to determine error types
func isConnectionError(err error) bool {
	// Check for common connection-related errors
	errStr := err.Error()
	return contains(errStr, "connection") ||
		contains(errStr, "timeout") ||
		contains(errStr, "network") ||
		contains(errStr, "dial")
}

func isAuthenticationError(err error) bool {
	// Check for authentication-related errors
	errStr := err.Error()
	return contains(errStr, "authentication") ||
		contains(errStr, "login") ||
		contains(errStr, "unauthorized") ||
		contains(errStr, "permission")
}

func isNotFoundError(err error) bool {
	// Check for not found errors
	errStr := err.Error()
	return contains(errStr, "not found") ||
		contains(errStr, "does not exist") || contains(errStr, "no snapshots for this VM")
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}
