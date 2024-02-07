package config

import (
	"crypto/tls"
	"log"
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
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Println(err)
		panic(err)
	}
	return cert
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
