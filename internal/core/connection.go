package core

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/blennster/gonnect/internal"
	"github.com/blennster/gonnect/internal/plugins"
	"github.com/blennster/gonnect/internal/security"
)

func pair(ctx context.Context, conn *tls.Conn, identity internal.GonnectIdentity, recv <-chan internal.ChanMsg) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case msg := <-recv:
		var pkt internal.GonnectPacket[any]
		err := json.Unmarshal(msg.Msg, &pkt)
		if err != nil {
			return err
		}

		var pairPkt internal.GonnectPacket[internal.GonnectPair]
		if pkt.Type == internal.GonnectPairType {
			err := json.Unmarshal(msg.Msg, &pairPkt)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown packet type: %q", msg.Msg)
		}

		if pairPkt.Body.Pair {
			ch := security.RequestPairApproval(identity.DeviceId)
			// The same broker is used for all connections,
			// therefore it needs to be checked in a loop
			for {
				select {
				case <-ctx.Done():
					return errors.New("context done")
				case approval := <-ch:
					// make sure that it is this device it is trying to pair with
					if approval {
						pkt := internal.NewGonnectPacket(internal.GonnectPair{Pair: true})
						b, _ := json.Marshal(pkt)
						_, err := conn.Write(append(b, '\n'))
						if err != nil {
							// r.Reply(fmt.Sprintf("pairing failed with message %q", err))
							return err
						}

						// r.Reply(identity.DeviceId)
						security.Devices.Add(identity.DeviceId, conn.ConnectionState().PeerCertificates[0])
						return nil
					}
				}
			}
		} else {
			security.Devices.Remove(identity.DeviceId)
			slog.Info("unpairing", "with", identity.DeviceId)
			return errors.New("unpairing")
		}
	}
}

func Handle(ctx context.Context, s *tls.Conn, identity internal.GonnectIdentity) {

	// Read from a connection in another goroutine to be able to sync everything
	// The buffer should be handled with care as it is not thread safe, but can be
	// handled by calling the function after buffer processing is done
	recv := make(chan internal.ChanMsg)
	go func() {
		buf := [1024 * 4]byte{}
		for {
			n, err := s.Read(buf[:])
			recv <- internal.NewChanMsg(buf[:n], err)
			if err != nil {
				return
			}
		}
	}()

	savedCert := security.Devices.Get(identity.DeviceId)
	if savedCert == nil {
		err := pair(ctx, s, identity, recv)
		if err != nil {
			slog.ErrorContext(ctx, "failed to pair", "to", identity.DeviceId, "error", err)
			return
		}
	}

	// Cert may have been updated if we paired
	savedCert = security.Devices.Get(identity.DeviceId)
	var pluginCh <-chan plugins.GonnectPluginMessage
	if savedCert.Equal(s.ConnectionState().PeerCertificates[0]) {
		ctx, pluginCh = plugins.WithPlugins(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			return
		// The plugin ch is for when a plugin wants to notify a client device about something
		case msg := <-pluginCh:
			if msg.Err != nil {
				slog.ErrorContext(ctx, "plugin error", "error", msg.Err)
				return
			}

			slog.Debug("writing to", "to", identity.DeviceId, "data", string(msg.Msg))
			_, err := s.Write(append(msg.Msg, '\n'))
			if err != nil {
				slog.ErrorContext(ctx, "failed to send data", "device", identity.DeviceId, "error", err)
				return
			}
		// We received a message from the client
		case msg := <-recv:
			err := msg.Err
			if err != nil {
				// connection was terminated or the server has closed
				if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
					return
				}
				slog.ErrorContext(ctx, "failed to read data", "device", identity.DeviceId, "error", err)
				return
			}

			var pkt internal.GonnectPacket[any]
			err = json.Unmarshal(msg.Msg, &pkt)
			if err != nil {
				slog.Error("failed to unmarshal", "device", identity.DeviceId, "error", err, "msg", msg.Msg)
				return
			}

			switch pkt.Type {
			case internal.GonnectPairType:
				var pkt internal.GonnectPacket[internal.GonnectPair]
				err := json.Unmarshal(msg.Msg, &pkt)
				if err != nil {
					slog.Error("failed to unmarshal", "device", identity.DeviceId, "error", err)
					return
				}
				if !pkt.Body.Pair {
					security.Devices.Remove(identity.DeviceId)
					slog.Info("unpairing", "device", identity.DeviceId)
					return
				}
			default:
				savedCert := security.Devices.Get(identity.DeviceId)
				if savedCert == nil || !savedCert.Equal(s.ConnectionState().PeerCertificates[0]) {
					slog.Warn("untrusted device tried to communicate", "device", identity.DeviceId)
					return
				}

				resp := plugins.Handle(ctx, msg.Msg)
				if resp != nil {
					// Make sure to include the newline :S
					s.Write(append(resp, '\n'))
				}
			}
		}
	}
}
