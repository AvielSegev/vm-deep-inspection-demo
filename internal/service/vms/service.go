package vms

import (
	"context"

	"github.com/nirarg/vm-deep-inspection-demo/internal/vmware"
	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
)

// Service provides VM discovery and management functionality
type Service struct {
	client *vmware.Client
	logger *logrus.Logger
}

// NewService creates a new VM service instance
func NewService(client *vmware.Client, logger *logrus.Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

// getGovmomiClient returns the underlying govmomi client
func (s *Service) getGovmomiClient(ctx context.Context) (*govmomi.Client, error) {
	return s.client.GetClient(ctx)
}

// getDefaultDatacenter is a helper to get the default datacenter
func (s *Service) getDefaultDatacenter(ctx context.Context, finder *find.Finder) (*object.Datacenter, error) {
	datacenter, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		return nil, err
	}
	finder.SetDatacenter(datacenter)
	return datacenter, nil
}

// findVMByName is a helper to find a VM by name
func (s *Service) findVMByName(ctx context.Context, name string) (*object.VirtualMachine, *object.Datacenter, error) {
	client, err := s.getGovmomiClient(ctx)
	if err != nil {
		return nil, nil, err
	}

	finder := find.NewFinder(client.Client, true)

	datacenter, err := s.getDefaultDatacenter(ctx, finder)
	if err != nil {
		return nil, nil, err
	}

	vm, err := finder.VirtualMachine(ctx, name)
	if err != nil {
		return nil, nil, err
	}

	return vm, datacenter, nil
}
