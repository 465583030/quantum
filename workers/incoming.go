package workers

import (
	"encoding/binary"
	"github.com/Supernomad/quantum/agg"
	"github.com/Supernomad/quantum/backend"
	"github.com/Supernomad/quantum/common"
	"github.com/Supernomad/quantum/inet"
	"github.com/Supernomad/quantum/socket"
)

// Incoming external packet interface which handles reading packets off of a Socket object
type Incoming struct {
	cfg        *common.Config
	aggregator *agg.Agg
	tunnel     inet.Interface
	sock       socket.Socket
	store      backend.Backend
	stop       bool
}

func (incoming *Incoming) resolve(payload *common.Payload) (*common.Payload, *common.Mapping, bool) {
	dip := binary.LittleEndian.Uint32(payload.IPAddress)

	if mapping, ok := incoming.store.GetMapping(dip); ok {
		return payload, mapping, true
	}

	return nil, nil, false
}

func (incoming *Incoming) unseal(payload *common.Payload, mapping *common.Mapping) (*common.Payload, bool) {
	_, err := mapping.Cipher.Open(payload.Packet[:0], payload.Nonce, payload.Packet, nil)
	if err != nil {
		return nil, false
	}

	return payload, true
}

func (incoming *Incoming) stats(dropped bool, payload *common.Payload, mapping *common.Mapping) {
	aggData := &agg.AggData{
		Direction: agg.Incoming,
		Dropped:   dropped,
	}

	if payload != nil {
		aggData.Bytes += uint64(payload.Length)
	}

	if mapping != nil {
		aggData.PrivateIP = mapping.PrivateIP.String()
	}

	incoming.aggregator.Aggs <- aggData
}

func (incoming *Incoming) pipeline(buf []byte, queue int) bool {
	payload, ok := incoming.sock.Read(buf, queue)
	if !ok {
		incoming.stats(true, payload, nil)
		return ok
	}
	payload, mapping, ok := incoming.resolve(payload)
	if !ok {
		incoming.stats(true, payload, mapping)
		return ok
	}
	payload, ok = incoming.unseal(payload, mapping)
	if !ok {
		incoming.stats(true, payload, mapping)
		return ok
	}
	incoming.stats(false, payload, mapping)
	return incoming.tunnel.Write(payload, queue)
}

// Start handling packets
func (incoming *Incoming) Start(queue int) {
	go func() {
		buf := make([]byte, common.MaxPacketLength)
		for !incoming.stop {
			incoming.pipeline(buf, queue)
		}
	}()
}

// Stop handling packets
func (incoming *Incoming) Stop() {
	incoming.stop = true
}

// NewIncoming object
func NewIncoming(cfg *common.Config, aggregator *agg.Agg, store backend.Backend, tunnel inet.Interface, sock socket.Socket) *Incoming {
	return &Incoming{
		cfg:        cfg,
		aggregator: aggregator,
		tunnel:     tunnel,
		sock:       sock,
		store:      store,
		stop:       false,
	}
}
