package security

import (
	"crypto/x509"
	"os"

	"github.com/blennster/gonnect/internal/config"
)

// Unix file system is "thread safe" so locking is not needed
type fileStore struct{}

// Add implements DeviceStore.
func (fileStore) Add(device string, cert *x509.Certificate) {
	pem := EncodePem(*cert)
	err := os.WriteFile(config.DataHome()+"/"+device+".pem", pem, 0600)

	if err != nil {
		panic(err)
	}
}

// Get implements DeviceStore.
func (fileStore) Get(device string) *x509.Certificate {
	f, err := os.ReadFile(config.DataHome() + "/" + device + ".pem")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		panic(err)
	}

	x509Cert, err := DecodePem(f)
	if err != nil {
		return nil
	}
	return x509Cert

}

// GetDiscoveredDevices implements DeviceStore.
func (fileStore) GetDiscoveredDevices() []string {
	os.ReadDir(config.DataHome())

	panic("unimplemented")
}

// Remove implements DeviceStore.
func (fileStore) Remove(device string) {
	err := os.Remove(config.DataHome() + "/" + device + ".pem")
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
	}
}
