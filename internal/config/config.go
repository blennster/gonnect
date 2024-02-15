package config

import (
	"crypto/tls"
	"os"

	"github.com/google/uuid"
)

func DataHome() string {
	if customHome, ok := os.LookupEnv("XDG_DATA_HOME"); ok {
		return customHome
	}

	// Maybe some runtime.GOOS check
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	_, err = os.Stat(homeDir + "/.local/share/gonnect")
	if err != nil {
		os.Mkdir(homeDir+"/.local/share/gonnect", 0700)
	}

	return homeDir + "/.local/share/gonnect"
}

func GetCert() tls.Certificate {
	certPath := DataHome() + "/cert.pem"
	keyPath := DataHome() + "/key.pem"

	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)

	if certErr != nil || keyErr != nil {
		GenerateCerts(GetName(), DataHome()+"/")
	}

	cert, certErr := os.ReadFile(certPath)
	key, keyErr := os.ReadFile(keyPath)

	if certErr == nil && keyErr == nil {
		cert, err := tls.X509KeyPair(cert, key)
		if err != nil {
			panic(err)
		}
		return cert
	}

	panic("failed to read certs")
}

func GetId() string {
	if contents, err := os.ReadFile(DataHome() + "/id"); err == nil {
		return string(contents)
	}

	id := uuid.New()
	os.WriteFile(DataHome()+"/id", []byte(id.String()), 0600)

	return id.String()
}

func GetName() string {
	hostname, _ := os.Hostname()
	return hostname
}

func GetType() string {
	// Simple battery check
	_, err := os.Stat("/sys/class/power_supply/BAT0")
	if err == nil {
		return "laptop"
	}

	return "desktop"
}
