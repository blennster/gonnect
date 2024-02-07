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
	currentClients := make(map[string]context.CancelFunc)
	buf := [4096]byte{}

	for {
		n, addr, err := listener.ReadFromUDP(buf[:])
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			panic(err)
		}
		if cancel, ok := currentClients[addr.String()]; ok {
			cancel()
			slog.Debug("duplicate connection, cancelling", "addr", addr)
		}

		ctx, cancel := context.WithCancel(baseCtx)
		currentClients[addr.String()] = cancel

		var data internal.GonnectPacket[internal.GonnectIdentity]
		err = json.Unmarshal(buf[:n], &data)
		if err != nil {
			slog.ErrorContext(ctx, "error while unmarshalling udp", err)
			cancel()
			delete(currentClients, addr.String())
			continue
		}

		go func(ctx context.Context, cancel context.CancelFunc) {
			target := netip.AddrPortFrom(addr.AddrPort().Addr(), data.Body.TcpPort)
			establishTcp(ctx, target, data.Body)
		}(ctx, cancel)
	}

	// Clean up
	for _, cancel := range currentClients {
		cancel()
	}
}
