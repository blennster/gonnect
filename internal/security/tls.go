package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"

	"github.com/blennster/gonnect/internal/config"
)

func GetCert() tls.Certificate {
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		panic(err)
	}
	return cert
}

func GetConfig() *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{GetCert()},
		ClientAuth:   tls.RequireAnyClientCert,
		ServerName:   config.GetId(),
	}

}

func EncodePem(cert x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
}

func DecodePem(cert []byte) (*x509.Certificate, error) {
	b, _ := pem.Decode(cert)
	if b == nil {
		return nil, fmt.Errorf("failed to decode pem")
	}

	if b.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("not a certificate: %s", b.Type)
	}

	return x509.ParseCertificate(b.Bytes)
}

func Upgrade(ctx context.Context, conn net.Conn, name string) (*tls.Conn, error) {
	config := GetConfig()
	// Android client hello does not send server name
	// config.GetConfigForClient = func(chi *tls.ClientHelloInfo) (*tls.Config, error) {
	// 	if chi.ServerName == name {
	// 		return nil, nil
	// 	}
	// 	return nil, fmt.Errorf("client name %s does not match with cerificate name %s", name, chi.ServerName)
	// }

	c := tls.Server(conn, config)
	err := c.HandshakeContext(ctx)
	return c, err
}
