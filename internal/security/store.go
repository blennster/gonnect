package security

import "crypto/x509"

type DeviceStore interface {
	Add(device string, cert *x509.Certificate)
	Remove(device string)
	Get(device string) *x509.Certificate
	// GetDiscoveredDevices() []string
}

var (
	// Devices DeviceStore = &inMemoryDeviceStore{devices: make(map[string]*x509.Certificate)}
	Devices DeviceStore = fileStore{}
)
