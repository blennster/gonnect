package discover

import (
	"context"
)

func Announce(ctx context.Context) {
	go AnnounceMdns(ctx)
	go ListenUdp(ctx)
	// go ListenTcp(ctx)
}
