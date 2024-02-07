package security

import (
	"crypto/x509"
	"log/slog"
	"sync"
)

type inMemoryDeviceStore struct {
	devices map[string]*x509.Certificate
	sync.RWMutex
}

func (d *inMemoryDeviceStore) Add(device string, cert *x509.Certificate) {
	d.Lock()
	defer d.Unlock()
	d.devices[device] = cert
}

func (d *inMemoryDeviceStore) Remove(device string) {
	d.Lock()
	defer d.Unlock()
	delete(d.devices, device)
}

func (d *inMemoryDeviceStore) Contains(device string) *x509.Certificate {
	d.RLock()
	defer d.Unlock()

	_, ok := d.devices[device]

	if ok {
		return d.devices[device]
	}

	slog.Warn("device name is in store but not no certificate", "device", device)
	return nil
}

func (d *inMemoryDeviceStore) GetDiscoveredDevices() []string {
	d.RLock()
	defer d.RUnlock()
	keys := make([]string, len(d.devices))
	for k := range d.devices {
		keys = append(keys, k)
	}

	return keys
}

func (d *inMemoryDeviceStore) GetDeviceCert(key string) *x509.Certificate {
	d.RLock()
	defer d.RUnlock()
	return d.devices[key]
}
