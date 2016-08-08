package main

import (
	"github.com/Supernomad/quantum/backend"
	"github.com/Supernomad/quantum/config"
	"github.com/Supernomad/quantum/socket"
	"github.com/Supernomad/quantum/tun"
	"github.com/Supernomad/quantum/workers"
	"github.com/go-playground/log"
	"github.com/go-playground/log/handlers/console"
	"os"
	"runtime"
	"strconv"
)

const version string = "0.6.0"

func handleError(err error) {
	if err != nil {
		os.Exit(1)
	}
}

func main() {
	cLog := console.New()
	log.RegisterHandler(cLog, log.AllLevels...)
	log.Infof("Starting up quantum v%s", version)

	cores := runtime.NumCPU()
	runtime.GOMAXPROCS(cores * 2)

	cfg := config.New()

	store, err := backend.New(cfg)
	handleError(err)

	err = store.Init()
	handleError(err)
	defer store.Stop()

	tunnel, err := tun.New(cfg.InterfaceName, cfg.PrivateIP, store.NetworkCfg, cores)
	handleError(err)
	defer tunnel.Close()

	sock, err := socket.New(cfg.ListenAddress, cfg.ListenPort, cores)
	handleError(err)
	defer sock.Close()

	outgoing := workers.NewOutgoing(cfg.PrivateIP, store, tunnel, sock)
	defer outgoing.Stop()

	incoming := workers.NewIncoming(cfg.PrivateIP, store, tunnel, sock)
	defer incoming.Stop()

	store.Start()
	for i := 0; i < cores; i++ {
		incoming.Start(i)
		outgoing.Start(i)
	}

	log.Info("Listening on TUN device:  ", tunnel.Name)
	log.Info("TUN network space:        ", store.NetworkCfg.Network)
	log.Info("TUN private IP address:   ", cfg.PrivateIP)
	log.Info("TUN public IP address:    ", cfg.PublicIP)
	log.Info("Listening on UDP address: ", cfg.ListenAddress+":"+strconv.Itoa(cfg.ListenPort))

	stop := make(chan bool)
	defer close(stop)
	<-stop
}
