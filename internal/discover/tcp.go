package discover

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/netip"

	"github.com/blennster/gonnect/internal"
	"github.com/blennster/gonnect/internal/core"
	"github.com/blennster/gonnect/internal/security"
)

func IdentityPacket() []byte {
	pkt := internal.NewGonnectPacket(internal.Identity())

	data, err := json.Marshal(pkt)
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')

	return data
}

func handleTcp(ctx context.Context, addr netip.AddrPort, identity internal.GonnectIdentity) {
	wg := internal.WgFromContext(ctx)
	defer wg.Done()

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr.String())
	if err != nil {
		slog.Error("failed to connect", "address", addr, "error", err)
		return
	}
	defer conn.Close()

	idPacket := IdentityPacket()
	slog.Debug("sending capabilities", "device", identity.DeviceId, "data", string(idPacket))
	_, err = conn.Write(idPacket)
	if err != nil {
		slog.Error("failed to send capabilities", "device", identity.DeviceId, "error", err)
		return
	}

	slog.Debug("upgrading to tls", "device", identity.DeviceId)
	s, err := security.Upgrade(ctx, conn, identity.DeviceId)
	if err != nil {
		slog.Error("failed to upgrade to tls", "address", addr, "error", err)
		return
	}
	slog.Debug("upgraded to tls", "device", identity.DeviceId)

	// Check that the device name is not spoofed
	savedCert := security.Devices.Get(identity.DeviceId)
	if savedCert != nil && !savedCert.Equal(s.ConnectionState().PeerCertificates[0]) {
		slog.Warn("name or certificate mismatch, dropping", "device", identity.DeviceId)
		return
	}

	core.Handle(ctx, s, identity)
}
