package discover

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/netip"

	"github.com/blennster/gonnect/internal"
	"github.com/blennster/gonnect/internal/config"
	"github.com/blennster/gonnect/internal/plugins"
	"github.com/blennster/gonnect/internal/security"
)

type chanMsg struct {
	Msg []byte
	Err error
}

func identityPacket() []byte {
	info := internal.GonnectIdentity{
		DeviceId:             config.GetId(),
		DeviceName:           config.GetName(),
		DeviceType:           config.GetType(),
		IncomingCapabilities: nil,
		OutgoingCapabilities: nil,
		ProtocolVersion:      internal.ProtocolVersion, // Magic value
		TcpPort:              0,                        // not used
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

func handleTcp(ctx context.Context, addr netip.AddrPort, identity internal.GonnectIdentity) {
	wg := internal.WgFromContext(ctx)
	defer wg.Done()

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
	s, err := security.Upgrade(ctx, conn, identity.DeviceId)
	if err != nil {
		slog.Error("failed to upgrade to tls", "for", identity.DeviceId, "error", err)
		return
	}
	slog.Debug("upgraded", "for", identity.DeviceId)

	// Check that the device name is not spoofed
	savedCert := security.Devices.Get(identity.DeviceId)
	if savedCert != nil && !savedCert.Equal(s.ConnectionState().PeerCertificates[0]) {
		slog.Info("name or certificate mismatch, dropping", "DeviceId", identity.DeviceId)
		return
	}

	var pluginCh <-chan []byte
	if savedCert.Equal(s.ConnectionState().PeerCertificates[0]) {
		ctx, pluginCh = plugins.WithPlugins(ctx)
	}

	buf := [1024 * 4]byte{}
	recv := make(chan chanMsg)

	// Read from a connection in another goroutine to be able to sync everything
	// The buffer should be handled with care as it is not thread safe, but can be
	// handled by calling the function after buffer processing is done
	recvFunc := func() {
		n, err := s.Read(buf[:])
		recv <- chanMsg{Msg: buf[:n], Err: err}
	}

	for {
		go recvFunc()
		select {
		case <-ctx.Done():
			return
		case msg := <-pluginCh:
			if msg != nil {
				slog.Debug("writing to", "to", identity.DeviceId, "data", string(msg))
				_, err := s.Write(append(msg, '\n'))
				if err != nil {
					slog.ErrorContext(ctx, "failed to write", "to", identity.DeviceId, "error", err)
					return
				}
			}
		case msg := <-recv:
			err := msg.Err
			if err != nil {
				// connection was terminated or the server has closed
				if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
					return
				}
				slog.ErrorContext(ctx, "failed to read", "from", identity.DeviceId, "error", err)
				return
			}

			var pkt internal.GonnectPacket[any]
			err = json.Unmarshal(msg.Msg, &pkt)
			if err != nil {
				slog.Error("failed to unmarshal", "from", identity.DeviceId, "error", err, "msg", msg.Msg)
				return
			}

			switch pkt.Type {
			case internal.GonnectPairType:
				var pkt internal.GonnectPacket[internal.GonnectPair]
				err := json.Unmarshal(msg.Msg, &pkt)
				if err != nil {
					slog.Error("failed to unmarshal", "from", identity.DeviceId, "error", err)
					return
				}

				if pkt.Body.Pair {
					slog.Info("pairing", "with", identity.DeviceId)
					security.Devices.Add(identity.DeviceId, s.ConnectionState().PeerCertificates[0])
					pkt := internal.NewGonnectPacket[internal.GonnectPair](internal.GonnectPair{Pair: true})
					b, _ := json.Marshal(pkt)
					s.Write(append(b, '\n'))
					s.Write(identityPacket())
					ctx, pluginCh = plugins.WithPlugins(ctx)

				} else {
					security.Devices.Remove(identity.DeviceId)
					slog.Info("unpairing", "with", identity.DeviceId)
					return
				}
			default:
				savedCert := security.Devices.Get(identity.DeviceId)
				if savedCert == nil || !savedCert.Equal(s.ConnectionState().PeerCertificates[0]) {
					slog.Info("untrusted device tried to communicate", "from", target)
					return
				}

				resp := plugins.Route(ctx, msg.Msg)
				if resp != nil {
					s.Write(append(resp, '\n'))
				}
			}
		}
	}
}
