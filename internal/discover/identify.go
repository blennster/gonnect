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
	"github.com/blennster/gonnect/internal/plugins"
	"github.com/blennster/gonnect/internal/security"
)

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

func pair(ctx context.Context, conn *tls.Conn, identity internal.GonnectIdentity, recv <-chan internal.ChanMsg) error {
	select {
	case <-ctx.Done():
		return errors.New("context done")
	case msg := <-recv:
		var pkt internal.GonnectPacket[any]
		err := json.Unmarshal(msg.Msg, &pkt)
		if err != nil {
			// slog.Error("failed to unmarshal", "from", identity.DeviceId, "error", err, "msg", msg.Msg)
			return err
		}

		if pkt.Type == internal.GonnectPairType {
			var pkt internal.GonnectPacket[internal.GonnectPair]
			err := json.Unmarshal(msg.Msg, &pkt)
			if err != nil {
				// slog.Error("failed to unmarshal", "from", identity.DeviceId, "error", err)
				return err
			}

			if pkt.Body.Pair {
				// slog.Info("pairing", "with", identity.DeviceId)
				security.Devices.Add(identity.DeviceId, conn.ConnectionState().PeerCertificates[0])
				pkt := internal.NewGonnectPacket[internal.GonnectPair](internal.GonnectPair{Pair: true})

				r := internal.GetRpc(ctx)

				slog.Info("pair requested", "from", identity.DeviceId)
				m := <-r.PairCh
				slog.Info("pair accepted", "for", m)

				if m == identity.DeviceId {
					r.PairCh <- "ok"
					b, _ := json.Marshal(pkt)
					conn.Write(append(b, '\n'))
					conn.Write(identityPacket())
					return nil
				}
				r.PairCh <- "ko"

			} else {
				security.Devices.Remove(identity.DeviceId)
				slog.Info("unpairing", "with", identity.DeviceId)
				return errors.New("unpairing")
			}
		}

		return errors.New("got non pair packet from unpaired device")
	}
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

	buf := [1024 * 4]byte{}
	recv := make(chan internal.ChanMsg)

	// Read from a connection in another goroutine to be able to sync everything
	// The buffer should be handled with care as it is not thread safe, but can be
	// handled by calling the function after buffer processing is done
	go func() {
		for {
			n, err := s.Read(buf[:])
			recv <- internal.NewChanMsg(buf[:n], err)
			if err != nil {
				return
			}
		}
	}()

	if savedCert == nil {
		err = pair(ctx, s, identity, recv)
		if err != nil {
			slog.ErrorContext(ctx, "failed to pair", "to", identity.DeviceId, "error", err)
			return
		}
	}

	var pluginCh <-chan plugins.GonnectPluginMessage
	if savedCert.Equal(s.ConnectionState().PeerCertificates[0]) {
		ctx, pluginCh = plugins.WithPlugins(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-pluginCh:
			if msg.Err != nil {
				slog.ErrorContext(ctx, "plugin error", "error", msg.Err)
				return
			}

			slog.Debug("writing to", "to", identity.DeviceId, "data", string(msg.Msg))
			_, err := s.Write(append(msg.Msg, '\n'))
			if err != nil {
				slog.ErrorContext(ctx, "failed to write", "to", identity.DeviceId, "error", err)
				return
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

				if !pkt.Body.Pair {
					security.Devices.Remove(identity.DeviceId)
					slog.Info("unpairing", "with", identity.DeviceId)
					return
				} else {
				}
			default:
				savedCert := security.Devices.Get(identity.DeviceId)
				if savedCert == nil || !savedCert.Equal(s.ConnectionState().PeerCertificates[0]) {
					slog.Info("untrusted device tried to communicate", "from", target)
					return
				}

				resp := plugins.Handle(ctx, msg.Msg)
				if resp != nil {
					s.Write(append(resp, '\n'))
				}
			}
		}
	}
}
