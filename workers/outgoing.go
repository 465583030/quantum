package workers

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"github.com/Supernomad/quantum/common"
	"github.com/Supernomad/quantum/logger"
	"github.com/Supernomad/quantum/socket"
	"github.com/Supernomad/quantum/tun"
	"net"
)

type Outgoing struct {
	tunnel     *tun.Tun
	sock       *socket.Socket
	privateIP  []byte
	privateKey *ecdsa.PrivateKey
	Mappings   map[uint32]*common.Mapping
	quit       chan bool
}

func (outgoing *Outgoing) Resolve(payload *common.Payload) (*common.Payload, *common.Mapping, bool) {
	dip := binary.LittleEndian.Uint32(payload.Packet[16:20])

	if mapping, ok := outgoing.Mappings[dip]; ok {
		copy(payload.IpAddress, outgoing.privateIP)
		return payload, mapping, true
	}

	return payload, nil, false
}

func (outgoing *Outgoing) Seal(payload *common.Payload, mapping *common.Mapping) (*common.Payload, bool) {
	_, err := rand.Read(payload.Nonce)
	if err != nil {
		return payload, false
	}

	mapping.Cipher.Seal(payload.Packet[:0], payload.Nonce, payload.Packet, nil)
	return payload, true
}

func (outgoing *Outgoing) Sign(payload *common.Payload) (*common.Payload, bool) {
	hash := sha256.Sum256(payload.Raw[common.PacketStart:payload.Length])

	r, s, err := ecdsa.Sign(rand.Reader, outgoing.privateKey, hash[:])
	if err != nil {
		return nil, false
	}

	copy(payload.R, r.Bytes())
	copy(payload.S, s.Bytes())

	return payload, true
}

func (outgoing *Outgoing) Start(queue int) {
	go func() {
		for {
			payload, ok := outgoing.tunnel.Read(queue)
			if !ok {
				continue
			}
			payload, mapping, ok := outgoing.Resolve(payload)
			if !ok {
				continue
			}
			payload, ok = outgoing.Seal(payload, mapping)
			if !ok {
				continue
			}
			payload, ok = outgoing.Sign(payload)
			if !ok {
				continue
			}
			outgoing.sock.Write(payload, mapping.Sockaddr, queue)
		}
	}()
}

func (outgoing *Outgoing) Stop() {
	go func() {
		outgoing.quit <- true
	}()
}

func NewOutgoing(log *logger.Logger, privateIP string, privateKey *ecdsa.PrivateKey, mappings map[uint32]*common.Mapping, tunnel *tun.Tun, sock *socket.Socket) *Outgoing {
	return &Outgoing{
		tunnel:     tunnel,
		sock:       sock,
		privateIP:  net.ParseIP(privateIP).To4(),
		privateKey: privateKey,
		Mappings:   mappings,
		quit:       make(chan bool),
	}
}
