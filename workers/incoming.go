package workers

import (
	"encoding/binary"
	"github.com/Supernomad/quantum/backend"
	"github.com/Supernomad/quantum/common"
	"github.com/Supernomad/quantum/socket"
)

// Incoming external packet interface which handles reading packets off of a Socket object
type Incoming struct {
	tunnel     socket.Socket
	sock       socket.Socket
	store      *backend.Backend
	quit       chan bool
	QueueStats []*common.Stats
}

func (incoming *Incoming) resolve(payload *common.Payload) (*common.Payload, *common.Mapping, bool) {
	dip := binary.LittleEndian.Uint32(payload.IPAddress)

	if mapping, ok := incoming.store.GetMapping(dip); ok {
		return payload, mapping, true
	}

	return payload, nil, false
}

func (incoming *Incoming) unseal(payload *common.Payload, mapping *common.Mapping) (*common.Payload, bool) {
	_, err := mapping.Cipher.Open(payload.Packet[:0], payload.Nonce, payload.Packet, nil)
	if err != nil {
		return payload, false
	}

	return payload, true
}

func (incoming *Incoming) stats(payload *common.Payload, mapping *common.Mapping, queue int) {
	incoming.QueueStats[queue].Packets++
	incoming.QueueStats[queue].Bytes += uint64(payload.Length)

	if link, ok := incoming.QueueStats[queue].Links[mapping.PrivateIP]; !ok {
		incoming.QueueStats[queue].Links[mapping.PrivateIP] = &common.Stats{
			Packets: 1,
			Bytes:   uint64(payload.Length),
		}
	} else {
		link.Packets++
		link.Bytes += uint64(payload.Length)
	}
}

// Start handling packets
func (incoming *Incoming) Start(queue int) {
	go func() {
		buf := make([]byte, common.MaxPacketLength)
		for {
			payload, ok := incoming.sock.Read(buf, queue)
			if !ok {
				continue
			}
			payload, mapping, ok := incoming.resolve(payload)
			if !ok {
				continue
			}
			payload, ok = incoming.unseal(payload, mapping)
			if !ok {
				continue
			}
			incoming.stats(payload, mapping, queue)
			incoming.tunnel.Write(payload, mapping, queue)
		}
	}()
}

// Stop handling packets
func (incoming *Incoming) Stop() {
	go func() {
		incoming.quit <- true
	}()
}

// NewIncoming object
func NewIncoming(privateIP string, numWorkers int, store *backend.Backend, tunnel socket.Socket, sock socket.Socket) *Incoming {
	stats := make([]*common.Stats, numWorkers)
	for i := 0; i < numWorkers; i++ {
		stats[i] = common.NewStats()
	}
	return &Incoming{
		tunnel:     tunnel,
		sock:       sock,
		store:      store,
		quit:       make(chan bool),
		QueueStats: stats,
	}
}
