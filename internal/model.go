package internal

import "time"

type GonnectMessageType string

type GonnectPacketType interface {
	Type() GonnectMessageType
}

const (
	GonnectPingType      GonnectMessageType = "kdeconnect.ping"
	GonnectPairType                         = "kdeconnect.pair"
	GonnectClipboardType                    = "kdeconnect.clipboard"
	GonnectIdentityType                     = "kdeconnect.identity"
)

const (
	ProtocolVersion = 7
)

type GonnectPacket[T any] struct {
	Id   int64              `json:"id"`
	Type GonnectMessageType `json:"type"`
	Body T                  `json:"body"`
}

type GonnectIdentity struct {
	DeviceId             string   `json:"deviceId"`
	DeviceName           string   `json:"deviceName"`
	DeviceType           string   `json:"deviceType"`
	IncomingCapabilities []string `json:"incomingCapabilities"`
	OutgoingCapabilities []string `json:"outgoingCapabilities"`
	ProtocolVersion      int      `json:"protocolVersion"`
	TcpPort              uint16   `json:"tcpPort"`
}

type GonnectPair struct {
	Pair bool `json:"pair"`
}

type GonnectPing struct {
	Message *string `json:"message"`
}

func (GonnectIdentity) Type() GonnectMessageType {
	return GonnectIdentityType
}

func (GonnectPair) Type() GonnectMessageType {
	return GonnectPairType
}

func NewGonnectPacket[T GonnectPacketType](body T) GonnectPacket[T] {
	return GonnectPacket[T]{
		Id:   time.Now().Unix(),
		Type: GonnectMessageType(body.Type()),
		Body: body,
	}
}
