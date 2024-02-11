package internal

import (
	"encoding/json"
	"time"

	"github.com/blennster/gonnect/internal/config"
)

type GonnectMessageType string

type GonnectPacketType interface {
	Type() GonnectMessageType
}

const (
	GonnectPingType             = GonnectMessageType("kdeconnect.ping")
	GonnectPairType             = GonnectMessageType("kdeconnect.pair")
	GonnectClipboardType        = GonnectMessageType("kdeconnect.clipboard")
	GonnectClipboardConnectType = GonnectMessageType("kdeconnect.clipboard.connect")
	GonnectIdentityType         = GonnectMessageType("kdeconnect.identity")
)

const (
	ProtocolVersion = 7
)

type GonnectPacket[T any] struct {
	Id   int64              `json:"id"`
	Type GonnectMessageType `json:"type"`
	Body T                  `json:"body"`
}

func Infer[T any](pkt GonnectPacket[any]) (*T, error) {
	b, err := json.Marshal(pkt.Body)
	if err != nil {
		return nil, err
	}
	var value T
	err = json.Unmarshal(b, &value)
	if err != nil {
		return nil, err
	}
	return &value, err
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

type GonnectClipboard struct {
	Content string `json:"content"`
}

type GonnectClipboardConnect struct {
	GonnectClipboard
	Timestamp int `json:"timestamp"`
}

func (GonnectIdentity) Type() GonnectMessageType {
	return GonnectIdentityType
}

func (GonnectPair) Type() GonnectMessageType {
	return GonnectPairType
}

func (GonnectPing) Type() GonnectMessageType {
	return GonnectPingType
}

func (GonnectClipboard) Type() GonnectMessageType {
	return GonnectClipboardType
}

func (GonnectClipboardConnect) Type() GonnectMessageType {
	return GonnectClipboardConnectType
}

func NewGonnectPacket[T GonnectPacketType](body T) GonnectPacket[T] {
	return GonnectPacket[T]{
		Id:   time.Now().Unix(),
		Type: GonnectMessageType(body.Type()),
		Body: body,
	}
}

func Identity() GonnectIdentity {
	identity := GonnectIdentity{
		DeviceId:             config.GetId(),
		DeviceName:           config.GetName(),
		DeviceType:           config.GetType(),
		IncomingCapabilities: nil,
		OutgoingCapabilities: nil,
		ProtocolVersion:      ProtocolVersion, // Magic value
		TcpPort:              0,               // not used
	}

	identity.IncomingCapabilities = []string{
		"kdeconnect.ping",
		"kdeconnect.clipboard",
		"kdeconnect.clipboard.connect",
	}
	identity.OutgoingCapabilities = []string{
		"kdeconnect.ping",
		"kdeconnect.clipboard",
		"kdeconnect.clipboard.connect",
	}

	return identity
}
