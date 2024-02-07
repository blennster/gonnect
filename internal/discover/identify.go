package discover

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/netip"

	"github.com/blennster/gonnect/internal"
	"github.com/blennster/gonnect/internal/config"
	"github.com/blennster/gonnect/internal/security"
)

func ListenTcp(ctx context.Context) {
	wg := internal.WgFromContext(ctx)
	defer wg.Done()

	listener, err := net.Listen("tcp", ":1716")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	slog.Info("Listening on :1716/tcp")

	config := security.GetConfig()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
					return
				}
				panic(err)
			}
			slog.Debug("Got tcp connection", "from", conn.RemoteAddr())

			go func() {
				defer conn.Close()
				defer slog.Info("Connection closed")

				conn = tls.Server(conn, config)
				for {
					bytes := identityPacket()
					_, err = conn.Write(bytes)
					if err != nil {
						if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
							return
						}
						panic(err)
					}
				}
			}()
		}
	}()

	<-ctx.Done()
	slog.Info("Shutting down TLS.")
}

func identityPacket() []byte {
	info := internal.GonnectIdentity{
		DeviceId:             config.GetId(),
		DeviceName:           config.GetName(),
		DeviceType:           config.GetType(),
		IncomingCapabilities: nil,
		OutgoingCapabilities: nil,
		ProtocolVersion:      7, // Magic value
		TcpPort:              0, // not used
	}

	info.IncomingCapabilities = []string{
		"kdeconnect.ping",
		"kdeconnect.clipboard",
		"kdeconnect.clipboard.connect",
	}
	info.OutgoingCapabilities = []string{
		"kdeconnect.ping",
		"kdeconnect.clipboard",
		"kdeconnect.clipboard.connect",
	}
	pkt := internal.NewGonnectPacket(info)

	data, err := json.Marshal(pkt)
	if err != nil {
		panic(err)
	}
	data = append(data, '\n')

	return data
}

func establishTcp(ctx context.Context, addr netip.AddrPort, identity internal.GonnectIdentity) {
	// target := netip.AddrPortFrom(addr, port)
	target := addr
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", target.String())
	if err != nil {
		slog.Error("failed to connect", "to", target, "error", err)
		return
	}
	defer conn.Close()

	idPacket := identityPacket()
	slog.Debug("sending capabilities", "to", identity.DeviceId, "data", string(idPacket))
	_, err = conn.Write(idPacket)
	if err != nil {
		slog.Error("failed to send", "to", identity.DeviceId, "error", err)
		return
	}

	slog.Debug("upgrading to tls", "for", identity.DeviceId)
	s, err := security.Upgrade(conn, "")
	if err != nil {
		slog.Error("failed to upgrade to tls", "for", identity.DeviceId, "error", err)
		return
	}
	slog.Debug("upgraded", "for", identity.DeviceId)

	savedCert := security.Devices.Get(identity.DeviceId)
	if savedCert != nil && !savedCert.Equal(s.ConnectionState().PeerCertificates[0]) {
		slog.Info("name or certificate mismatch, dropping", "DeviceId", identity.DeviceId)
		return
	}

	buf := [4096]byte{}
	for {
		n, err := s.Read(buf[:])
		if err != nil {
			// connection was terminated or the server has closed
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				return
			}
			slog.ErrorContext(ctx, "failed to read", "from", identity.DeviceId, "error", err)
			return
		}

		var pkt internal.GonnectPacket[any]
		err = json.Unmarshal(buf[:n], &pkt)
		if err != nil {
			slog.Error("Failed to unmarshal", "from", identity.DeviceId, "error", err)
			return
		}

		switch pkt.Type {
		case internal.GonnectPairType:
			var pkt internal.GonnectPacket[internal.GonnectPair]
			err := json.Unmarshal(buf[:n], &pkt)
			if err != nil {
				slog.Error("failed to unmarshal", "from", identity.DeviceId, "error", err)
				return
			}

			if pkt.Body.Pair {
				slog.Info("pairing", "with", identity.DeviceId)
				security.Devices.Add(identity.DeviceId, s.ConnectionState().PeerCertificates[0])
				pkt = internal.NewGonnectPacket[internal.GonnectPair](internal.GonnectPair{Pair: true})
				b, _ := json.Marshal(pkt)
				s.Write(append(b, '\n'))
				s.Write(identityPacket())
			} else {
				security.Devices.Remove(identity.DeviceId)
				slog.Info("unpairing", "with", identity.DeviceId)
			}
		default:
			savedCert := security.Devices.Get(identity.DeviceId)
			if savedCert == nil || !savedCert.Equal(s.ConnectionState().PeerCertificates[0]) {
				slog.Info("untrusted device tried to communicate", "from", target)
				return
			}

			slog.Info("unhandled packet", "from", target, "type", pkt.Type, "body", pkt.Body)
		}
	}
}
