// Copyright (c) 2016 Christian Saide <Supernomad>
// Licensed under the MPL-2.0, for details see https://github.com/Supernomad/quantum/blob/master/LICENSE

package workers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"net"
	"testing"
	"time"

	"github.com/Supernomad/quantum/agg"
	"github.com/Supernomad/quantum/common"
	"github.com/Supernomad/quantum/datastore"
	"github.com/Supernomad/quantum/device"
	"github.com/Supernomad/quantum/socket"
)

var (
	testMapping, testMappingUnencrypted *common.Mapping
	outgoing, outgoingUnencrypted       *Outgoing
	incoming, incomingUnencrypted       *Incoming
	store, storeUnencrypted             *datastore.Mock

	dev       device.Device
	sock      socket.Socket
	privateIP = "10.1.1.1"
)

func init() {
	ip := net.ParseIP("10.8.0.1")
	ipv6 := net.ParseIP("dead::beef")

	store = &datastore.Mock{}
	storeUnencrypted = &datastore.Mock{}
	dev, _ = device.New(device.MOCKDevice, nil)
	sock, _ = socket.New(socket.MOCKSocket, nil)

	key := make([]byte, 32)
	rand.Read(key)

	block, _ := aes.NewCipher(key)
	aesgcm, _ := cipher.NewGCM(block)

	testMapping = &common.Mapping{IPv4: ip, IPv6: ipv6, PublicKey: make([]byte, 32), Cipher: aesgcm}
	testMappingUnencrypted = &common.Mapping{IPv4: ip, IPv6: ipv6, Unencrypted: true, PublicKey: make([]byte, 32), Cipher: aesgcm}

	store.InternalMapping = testMapping
	storeUnencrypted.InternalMapping = testMappingUnencrypted

	aggregator := agg.New(
		common.NewLogger(common.NoopLogger),
		&common.Config{
			StatsRoute:   "/stats",
			StatsPort:    1099,
			StatsAddress: "127.0.0.1",
			NumWorkers:   1,
		})
	aggregator.Start()

	incoming = NewIncoming(&common.Config{NumWorkers: 1, PrivateIP: ip, IsIPv6Enabled: true, IsIPv4Enabled: true}, aggregator, store, dev, sock)
	incomingUnencrypted = NewIncoming(&common.Config{Unencrypted: true, NumWorkers: 1, PrivateIP: ip, IsIPv6Enabled: true, IsIPv4Enabled: true}, aggregator, storeUnencrypted, dev, sock)
	outgoing = NewOutgoing(&common.Config{NumWorkers: 1, PrivateIP: ip, IsIPv6Enabled: true, IsIPv4Enabled: true}, aggregator, store, dev, sock)
	outgoingUnencrypted = NewOutgoing(&common.Config{Unencrypted: true, NumWorkers: 1, PrivateIP: ip, IsIPv6Enabled: true, IsIPv4Enabled: true}, aggregator, storeUnencrypted, dev, sock)
}

func benchmarkEncryptedIncomingPipeline(buf []byte, queue int, b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		incoming.pipeline(buf, queue)
	}
}

func BenchmarkEncryptedIncomingPipeline(b *testing.B) {
	buf := make([]byte, common.MaxPacketLength)
	rand.Read(buf)

	payload := common.NewTunPayload(buf, common.MTU)
	if sealed, pass := outgoing.seal(payload, testMapping); pass {
		benchmarkEncryptedIncomingPipeline(sealed.Raw, 0, b)
	} else {
		panic("Seal failed something is wrong")
	}
}

func benchmarkUnencryptedIncomingPipeline(buf []byte, queue int, b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		incomingUnencrypted.pipeline(buf, queue)
	}
}

func BenchmarkUnencryptedIncomingPipeline(b *testing.B) {
	buf := make([]byte, common.MaxPacketLength)
	rand.Read(buf)

	payload := common.NewTunPayload(buf, common.MTU)
	if sealed, pass := outgoingUnencrypted.seal(payload, testMappingUnencrypted); pass {
		benchmarkUnencryptedIncomingPipeline(sealed.Raw, 0, b)
	} else {
		panic("Seal failed something is wrong")
	}
}

func TestIncomingPipeline(t *testing.T) {
	buf := make([]byte, common.MaxPacketLength)
	rand.Read(buf)

	payload := common.NewTunPayload(buf, common.MTU)
	if sealed, pass := outgoing.seal(payload, testMapping); pass {
		if !incoming.pipeline(sealed.Raw, 0) {
			panic("Somthing is wrong.")
		}
	} else {
		panic("Seal failed something is wrong")
	}
}

func TestIncoming(t *testing.T) {
	incoming.Start(0)
	time.Sleep(2 * time.Second)
	incoming.Stop()
}

func benchmarkEncryptedOutgoingPipeline(buf []byte, queue int, b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if !outgoing.pipeline(buf, queue) {
			panic("Somthing is wrong.")
		}
	}
}

func BenchmarkEncryptedOutgoingPipeline(b *testing.B) {
	buf := make([]byte, common.MaxPacketLength)
	rand.Read(buf)

	benchmarkEncryptedOutgoingPipeline(buf, 0, b)
}

func benchmarkUnencryptedOutgoingPipeline(buf []byte, queue int, b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if !outgoingUnencrypted.pipeline(buf, queue) {
			panic("Somthing is wrong.")
		}
	}
}

func BenchmarkUnencryptedOutgoingPipeline(b *testing.B) {
	buf := make([]byte, common.MaxPacketLength)
	rand.Read(buf)

	benchmarkUnencryptedOutgoingPipeline(buf, 0, b)
}

func TestOutgoingPipeline(t *testing.T) {
	buf := make([]byte, common.MaxPacketLength)
	rand.Read(buf)
	if !outgoing.pipeline(buf, 0) {
		panic("Somthing is wrong.")
	}
}

func TestOutgoing(t *testing.T) {
	outgoing.Start(0)
	time.Sleep(2 * time.Second)
	outgoing.Stop()
}
