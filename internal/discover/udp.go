package discover

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/netip"

	"github.com/blennster/gonnect/internal"
)

func ListenUdp(ctx context.Context) {
	wg := internal.WgFromContext(ctx)
	defer wg.Done()

	addr, _ := net.ResolveUDPAddr("udp", ":1716")
	listener, err := net.ListenUDP("udp", addr)
	slog.Info("listening on UDP", "addr", addr)

	if err != nil {
		panic(err)
	}

	go handleUdp(ctx, listener)

	<-ctx.Done()
	listener.Close()
	slog.Info("shutting down UDP listener.")
}

func handleUdp(baseCtx context.Context, listener *net.UDPConn) {
	currentClients := make(map[string]any)
	buf := [4096]byte{}

	for {
		n, addr, err := listener.ReadFromUDP(buf[:])
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			panic(err)
		}

		var identityPacket internal.GonnectPacket[internal.GonnectIdentity]
		err = json.Unmarshal(buf[:n], &identityPacket)
		if err != nil {
			slog.Error("error while unmarshalling udp", err)
			continue
		}

		if _, ok := currentClients[identityPacket.Body.DeviceId]; ok {
			slog.Debug("duplicate connection, dropping", "addr", addr)
			continue
		}
		ctx := internal.WithIdentity(baseCtx, identityPacket.Body)
		currentClients[identityPacket.Body.DeviceId] = nil

		go func(ctx context.Context) {
			target := netip.AddrPortFrom(addr.AddrPort().Addr(), identityPacket.Body.TcpPort)
			handleTcp(baseCtx, target, identityPacket.Body)
			delete(currentClients, identityPacket.Body.DeviceId)
		}(ctx)
	}
}
