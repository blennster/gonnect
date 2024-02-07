package plugins

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os/exec"

	"github.com/blennster/gonnect/internal"
)

// The clipboard plugin handles syncing the clipboard with the other device
// it is bidirectional with the desktop clipboard being authorative ish
type ClipboardPlugin struct {
	// synchronization channel used for not sending back the same data that was received
	syncCh chan struct{}
}

func NewClipboardPlugin(ctx context.Context, ch chan<- []byte) ClipboardPlugin {
	c := ClipboardPlugin{
		syncCh: make(chan struct{}, 1), // One buffer is needed to stop blocking
	}

	go c.clipboardWatcher(ctx, ch)

	return c
}

// React implements GonnectPlugin.
func (c ClipboardPlugin) React(ctx context.Context, data []byte) any {
	var pkt internal.GonnectPacket[internal.GonnectClipboard]
	err := json.Unmarshal(data, &pkt)
	if err != nil {
		panic(err)
	}

	c.syncCh <- struct{}{}
	cmd := exec.Command("wl-copy")
	cmd.Stdin = bytes.NewReader([]byte(pkt.Body.Content))
	cmd.Run()
	slog.Debug("wrote clipboard", "content", pkt.Body.Content)

	return internal.NewGonnectPacket[internal.GonnectClipboardConnect](internal.GonnectClipboardConnect{})
}

// listen for clipboard changes and send to other device when notified about change
func (c *ClipboardPlugin) clipboardWatcher(ctx context.Context, ch chan<- []byte) {
	slog.Debug("clipboard watcher started")

	cmd := exec.CommandContext(ctx, "wl-paste", "-t", "text", "-w", "tee")
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	defer func() {
		slog.Debug("killing clipboard watcher")
		err = cmd.Process.Kill()
		if err != nil {
			slog.Error("error when killing clipboard watcher", err)
		}
	}()

	buf := [4096]byte{}
	for {
		select {
		case <-ctx.Done():
			slog.Debug("killing clipboard watcher")
			return
		default:
			slog.Debug("waiting for read")
			n, err := pipe.Read(buf[:])
			if err != nil {
				slog.Error("error when reading in clipboard watcher", err)
				return
			}

			// only process the data if it comes from an external source
			select {
			case <-c.syncCh:
				continue
			default:
				slog.Debug("clipboard watcher", "data", buf[:n])
				pkt := internal.NewGonnectPacket[internal.GonnectClipboard](internal.GonnectClipboard{Content: string(buf[:n])})
				data, err := json.Marshal(pkt)
				if err != nil {
					panic(err)
				}
				ch <- data
			}
		}
	}
}
