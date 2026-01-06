package vmhandler

import (
	"github.com/kubev2v/vm-migration-detective/pkg/persistent"
	"github.com/nirarg/vm-deep-inspection-demo/internal/service/vms"
	"github.com/sirupsen/logrus"
)

// VMHandler handles VM-related API requests
type VMHandler struct {
	vmService *vms.Service
	vmClient  *vms.Client
	inspector *persistent.Inspector
	logger    *logrus.Logger
}

// NewVMHandler creates a new VM handler instance
func NewVMHandler(vmService *vms.Service, vmClient *vms.Client, inspector *persistent.Inspector, logger *logrus.Logger) *VMHandler {
	return &VMHandler{
		vmService: vmService,
		vmClient:  vmClient,
		inspector: inspector,
		logger:    logger,
	}
}
