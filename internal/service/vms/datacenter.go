package vms

import "context"

// GetDatacenterName returns the datacenter name for a given VM
func (s *Service) GetDatacenterName(ctx context.Context, vmName string) (string, error) {
	_, datacenter, err := s.findVMByName(ctx, vmName)
	if err != nil {
		return "", err
	}
	return datacenter.Name(), nil
}
