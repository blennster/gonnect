
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
