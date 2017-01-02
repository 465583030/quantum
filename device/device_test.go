// Copyright (c) 2016 Christian Saide <Supernomad>
// Licensed under the MPL-2.0, for details see https://github.com/Supernomad/quantum/blob/master/LICENSE

package device

import (
	"math/rand"
	"net"
	"sync"
	"syscall"
	"testing"

	"github.com/Supernomad/quantum/common"
	"golang.org/x/net/ipv4"
)

var mutex = &sync.Mutex{}
var tun Device

func benchmarkWrite(payload *common.Payload, queue int, b *testing.B) {
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		if !tun.Write(payload, queue) {
			b.Fatal("Failed to write")
		}
	}
}

func BenchmarkWrite(b *testing.B) {
	mutex.Lock()

	if tun == nil {
		cfg := &common.Config{
			NumWorkers:    1,
			DeviceName:    "quantum%d",
			PrivateIP:     net.ParseIP("10.99.0.1"),
			NetworkConfig: common.DefaultNetworkConfig,
			ReuseFDS:      false,
		}
		var err error
		tun, err = New(TUNDevice, cfg)

		if err != nil {
			panic(err.Error())
		}
	}

	mutex.Unlock()

	buf := make([]byte, common.MaxPacketLength)
	rand.Read(buf)

	iph := &ipv4.Header{
		Version:  4,
		Len:      20,
		TOS:      0,
		TotalLen: common.MaxPacketLength,
		ID:       0,
		Flags:    0,
		FragOff:  0,
		TTL:      128,
		Protocol: syscall.ETH_P_IP,
		Checksum: 0,
		Src:      net.ParseIP("127.0.0.2"),
		Dst:      net.ParseIP("127.0.0.3"),
		Options:  nil,
	}
	payload := common.NewSockPayload(buf, common.MTU)
	iphBuf, _ := iph.Marshal()
	copy(payload.Packet[0:20], iphBuf)

	benchmarkWrite(payload, 0, b)
}
