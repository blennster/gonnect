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
type clipboardPlugin struct {
	// synchronization channel used for not sending back the same data that was received
	syncCh    chan struct{}
	connected bool
}

// Create a new clipboard plugin instance and start the clipboard watching goroutine
func NewClipboardPlugin(ctx context.Context, ch chan<- GonnectPluginMessage) *clipboardPlugin {
	c := clipboardPlugin{
		syncCh: make(chan struct{}, 1), // One buffer is needed to stop blocking
	}
	go c.clipboardWatcher(ctx, ch)

	data, err := json.Marshal(internal.NewGonnectPacket[internal.GonnectClipboardConnect](internal.GonnectClipboardConnect{}))
	if err != nil {
		panic(err)
	}
	// We dont need the NewChanMsg here since the slice does not live longer than this function
	ch <- GonnectPluginMessage{Msg: data}

	return &c
}

// React implements GonnectPlugin.
func (c *clipboardPlugin) React(ctx context.Context, data []byte) any {
	var packet internal.GonnectPacket[any]
	err := json.Unmarshal(data, &packet)
	if err != nil {
		panic(err)
	}

	returnPacket := internal.NewGonnectPacket[internal.GonnectClipboardConnect](internal.GonnectClipboardConnect{})

	if packet.Type == internal.GonnectClipboardConnectType {
		c.connected = true
		slog.Debug("cliboard connected")
		return returnPacket
	}

	var pkt internal.GonnectPacket[internal.GonnectClipboard]
	err = json.Unmarshal(data, &pkt)
	if err != nil {
		panic(err)
	}

	c.syncCh <- struct{}{}
	cmd := exec.Command("wl-copy")
	cmd.Stdin = bytes.NewReader([]byte(pkt.Body.Content))
	cmd.Run()
	slog.Debug("wrote clipboard", "content", pkt.Body.Content)

	return returnPacket
}

// listen for clipboard changes and send to other device when notified about change
func (c *clipboardPlugin) clipboardWatcher(ctx context.Context, ch chan<- GonnectPluginMessage) {
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

	// Read from the pipe in another goroutine
	// so that we can listen to the context done
	procCh := make(chan internal.ChanMsg)
	go func() {
		buf := [4096]byte{}
		for {
			n, err := pipe.Read(buf[:])
			if err != nil {
				slog.Error("error when reading in clipboard watcher", err)
				procCh <- internal.NewChanMsg(nil, err)
				return
			}
			procCh <- internal.NewChanMsg(buf[:n], nil)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-procCh:
			select {
			// Discard if it is a duplicate from us writing to the clipboard
			case <-c.syncCh:
			default:
				if msg.Err != nil {
					slog.Error("error when reading in clipboard watcher", msg.Err)
					ch <- GonnectPluginMessage(internal.NewChanMsg(nil, msg.Err))
					return
				}

				pkt := internal.NewGonnectPacket[internal.GonnectClipboard](internal.GonnectClipboard{Content: string(msg.Msg)})
				data, err := json.Marshal(pkt)
				if err != nil {
					panic(err)
				}
				if !c.connected {
					slog.Debug("not sending clipboard", "reason", "not connected")
					continue
				}

				slog.Debug("sending clipboard", "data", string(msg.Msg))
				ch <- GonnectPluginMessage(internal.NewChanMsg(data, nil))
			}
		}
	}
}
