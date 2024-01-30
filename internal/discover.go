package internal

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/netip"
	"sync"
	"time"
)

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
	Id   int64  `json:"id"`
	Type string `json:"type"`
	Body struct {
		Pair bool `json:"pair"`
	}
}

type GonnectPacket struct {
	Id   int64           `json:"id"`
	Type string          `json:"type"`
	Body GonnectIdentity `json:"body"`
}

func GetCert() tls.Certificate {
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Println(err)
		panic(err)
	}
	return cert
}

func ListenTcp(shutdown chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	listener, err := net.Listen("tcp", ":1716")
	if err != nil {
		log.Println(err)
		panic(err)
	}
	defer listener.Close()
	log.Println("Listening on :1716/tcp")

	config := tls.Config{Certificates: []tls.Certificate{GetCert()}}
	go func() {
		for {
			conn, err := listener.Accept()
			log.Println("New connection")
			if err != nil {
				log.Println(err)
				if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
					return
				}
				panic(err)
			}

			go func() {
				defer conn.Close()
				defer log.Println("Connection closed")

				conn = tls.Server(conn, &config)
				for {
					bytes := Response()
					_, err = conn.Write(bytes)
					if err != nil {
						if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
							return
						}
						log.Println(err)
						panic(err)
					}
				}
			}()
		}
	}()

	<-shutdown
	log.Println("Shutting down TLS.")
}

func Response() []byte {
	info := GonnectPacket{}
	info.Id = time.Now().Unix()
	info.Type = "kdeconnect.identity"
	// info.Body.DeviceId = "_7b6e5ae5_1ef5_4a20_84a5_06c200037ef3_"
	info.Body.DeviceId = "gonnect"
	info.Body.DeviceName = "16ach-gonnect"
	info.Body.DeviceType = "laptop"
	info.Body.ProtocolVersion = 7
	info.Body.TcpPort = 1716
	info.Body.IncomingCapabilities = []string{
		"kdeconnect.findmyphone.request",
		"kdeconnect.telephony",
		"kdeconnect.telephony.request_mute",
		"kdeconnect.ping",
		"kdeconnect.lock",
		"kdeconnect.share.request",
		"kdeconnect.systemvolume.request",
		"kdeconnect.battery",
		"kdeconnect.notification.request",
		"kdeconnect.notification",
		"kdeconnect.mousepad.echo",
		"kdeconnect.mpris.request",
		"kdeconnect.presenter",
		"kdeconnect.clipboard.connect",
		"kdeconnect.mpris",
		"kdeconnect.photo",
		"kdeconnect.sms.messages",
		"kdeconnect.sms.attachment_file",
		"kdeconnect.virtualmonitor.request",
		"kdeconnect.mousepad.keyboardstate",
		"kdeconnect.virtualmonitor",
		"kdeconnect.sftp",
		"kdeconnect.runcommand.request",
		"kdeconnect.contacts.response_uids_timestamps",
		"kdeconnect.runcommand",
		"kdeconnect.lock.request",
		"kdeconnect.systemvolume",
		"kdeconnect.battery.request",
		"kdeconnect.mousepad.request",
		"kdeconnect.clipboard",
		"kdeconnect.contacts.response_vcards",
		"kdeconnect.connectivity_report",
		"kdeconnect.bigscreen.stt",
	}
	info.Body.OutgoingCapabilities = []string{
		"kdeconnect.findmyphone.request",
		"kdeconnect.telephony",
		"kdeconnect.telephony.request_mute",
		"kdeconnect.ping",
		"kdeconnect.photo.request",
		"kdeconnect.contacts.request_all_uids_timestamps",
		"kdeconnect.sms.request_conversation",
		"kdeconnect.lock",
		"kdeconnect.share.request",
		"kdeconnect.systemvolume.request",
		"kdeconnect.notification.reply",
		"kdeconnect.sms.request",
		"kdeconnect.battery",
		"kdeconnect.notification",
		"kdeconnect.notification.request",
		"kdeconnect.contacts.request_vcards_by_uid",
		"kdeconnect.mpris.request",
		"kdeconnect.clipboard.connect",
		"kdeconnect.mpris",
		"kdeconnect.share.request.update",
		"kdeconnect.virtualmonitor.request",
		"kdeconnect.mousepad.keyboardstate",
		"kdeconnect.virtualmonitor",
		"kdeconnect.runcommand.request",
		"kdeconnect.sms.request_attachment",
		"kdeconnect.sms.request_conversations",
		"kdeconnect.connectivity_report.request",
		"kdeconnect.runcommand",
		"kdeconnect.lock.request",
		"kdeconnect.systemvolume",
		"kdeconnect.notification.action",
		"kdeconnect.battery.request",
		"kdeconnect.mousepad.request",
		"kdeconnect.clipboard",
		"kdeconnect.bigscreen.stt",
		"kdeconnect.sftp.request",
	}

	data, err := json.Marshal(info)
	if err != nil {
		log.Println(err)
		panic(err)
	}
	data = append(data, '\n')

	return data
}

func SendInfo(addr netip.Addr, port uint16, identity GonnectIdentity) (string, error) {
	c := &tls.Config{
		Certificates: []tls.Certificate{GetCert()},
		ClientAuth:   tls.RequireAnyClientCert,
		ServerName:   "_7b6e5ae5_1ef5_4a20_84a5_06c200037ef3_",
	}

	target := netip.AddrPortFrom(addr, port)
	conn, err := net.DialTCP("tcp", nil, net.TCPAddrFromAddrPort(target))
	if err != nil {
		if !errors.Is(err, io.EOF) {
			log.Println(err)
			panic(err)
		}
		return "", errors.New("Connection refused")
	}
	data := Response()
	log.Printf("Writing to %s\n", target)
	_, err = conn.Write(data)
	if err != nil {
		return "", err
	}

	log.Println("Upgrading to tls")
	s := tls.Server(conn, c)
	defer s.Close()
	err = s.Handshake()
	if err != nil {
		return "", err
	}
	log.Println("Upgraded")

	buf := make([]byte, 4096)
	s.Read(buf)
	log.Println(string(buf))

	return string(data), err
}

func ListenUdp(shutdown chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	addr, _ := net.ResolveUDPAddr("udp", ":1716")
	listener, err := net.ListenUDP("udp", addr)
	defer listener.Close()

	if err != nil {
		panic(err)
	}

	go func() {
		currentClients := make(map[string]struct{})
		for {
			buf := make([]byte, 4096)
			_, addr, err := listener.ReadFromUDP(buf)
			if err != nil {
				// Expected behaviour when shutting down
				if errors.Is(err, net.ErrClosed) {
					return
				}
				panic(err)
			}
			if _, ok := currentClients[addr.String()]; ok {
				log.Println("Already connected to, stopping", addr.String())
				continue
			}
			currentClients[addr.String()] = struct{}{}

			buf = bytes.Trim(buf, "\x00")

			var data GonnectPacket
			err = json.Unmarshal(buf, &data)
			if err != nil {
				log.Println("Json error:", err)
				continue
			}

			log.Printf("Received data from %s\n", addr.String())
			conn, _ := net.DialUDP("udp", nil, addr)
			_, err = conn.Write(Response())
			if err != nil {
				log.Println("Error while responding:", err)
			}
			conn.Close()

			_, err = SendInfo(addr.AddrPort().Addr(), data.Body.TcpPort, data.Body)
			if err != nil {
				log.Println("Error while establishing tcp", err)
			}
			delete(currentClients, addr.String())
		}
	}()

	<-shutdown
	log.Println("Shutting down udp.")
}
