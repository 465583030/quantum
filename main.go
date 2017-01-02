// Copyright (c) 2016 Christian Saide <Supernomad>
// Licensed under the MPL-2.0, for details see https://github.com/Supernomad/quantum/blob/master/LICENSE

package main

import (
	"os"

	"github.com/Supernomad/quantum/agg"
	"github.com/Supernomad/quantum/common"
	"github.com/Supernomad/quantum/datastore"
	"github.com/Supernomad/quantum/device"
	"github.com/Supernomad/quantum/socket"
	"github.com/Supernomad/quantum/workers"
)

func handleError(log *common.Logger, err error) {
	if err != nil {
		log.Error.Println(err.Error())
		os.Exit(1)
	}
}

func main() {
	log := common.NewLogger(common.InfoLogger)

	cfg, err := common.NewConfig(log)
	handleError(log, err)

	store, err := datastore.New(datastore.ETCDDatastore, log, cfg)
	handleError(log, err)

	err = store.Init()
	handleError(log, err)

	dev, err := device.New(device.TUNDevice, cfg)
	handleError(log, err)

	sock, err := socket.New(socket.UDPSocket, cfg)
	handleError(log, err)

	aggregator := agg.New(log, cfg)

	outgoing := workers.NewOutgoing(cfg, aggregator, store, dev, sock)
	incoming := workers.NewIncoming(cfg, aggregator, store, dev, sock)

	aggregator.Start()
	store.Start()
	for i := 0; i < cfg.NumWorkers; i++ {
		incoming.Start(i)
		outgoing.Start(i)
	}

	log.Info.Printf("[MAIN] Listening on TUN device:  %s", dev.Name())
	log.Info.Printf("[MAIN] TUN network space:        %s", cfg.NetworkConfig.Network)
	log.Info.Printf("[MAIN] TUN private IP address:   %s", cfg.PrivateIP)
	log.Info.Printf("[MAIN] TUN public IPv4 address:  %s", cfg.PublicIPv4)
	log.Info.Printf("[MAIN] TUN public IPv6 address:  %s", cfg.PublicIPv6)
	log.Info.Printf("[MAIN] Listening on UDP port:    %d", cfg.ListenPort)

	fds := make([]int, cfg.NumWorkers*2)
	copy(fds[0:cfg.NumWorkers], dev.Queues())
	copy(fds[cfg.NumWorkers:cfg.NumWorkers*2], sock.Queues())

	signaler := common.NewSignaler(log, cfg, fds, map[string]string{common.RealDeviceNameEnv: dev.Name()})

	err = signaler.Wait()
	handleError(log, err)

	aggregator.Stop()
	store.Stop()

	incoming.Stop()
	outgoing.Stop()

	sock.Close()
	dev.Close()
}
