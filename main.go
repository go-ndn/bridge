package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/go-ndn/log"
	"github.com/go-ndn/ndn"
	"github.com/go-ndn/packet"
)

var (
	configPath = flag.String("config", "bridge.json", "config path")
	debug      = flag.Bool("debug", false, "enable logging")
)

var (
	key ndn.Key
)

func main() {
	flag.Parse()

	// config
	configFile, err := os.Open(*configPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer configFile.Close()

	err = json.NewDecoder(configFile).Decode(&config)
	if err != nil {
		log.Fatalln(err)
	}

	// key
	pem, err := os.Open(config.PrivateKeyPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer pem.Close()

	key, err = ndn.DecodePrivateKey(pem)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("key", key.Locator())

	// local face
	local, err := newFace(config.Local.Network, config.Local.Address, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer local.Close()
	// remote face
	recv := make(chan *ndn.Interest)
	remote, err := newFace(config.Remote.Network, config.Remote.Address, recv)
	if err != nil {
		log.Fatalln(err)
	}
	defer remote.Close()

	go local.advertise(remote)

	// create remote tunnel
	for i := range recv {
		local.ServeNDN(remote, i)
	}
}

func newFace(network, address string, recv chan<- *ndn.Interest) (f *face, err error) {
	conn, err := packet.Dial(network, address)
	if err != nil {
		return
	}
	f = &face{
		Face: ndn.NewFace(conn, recv),
	}
	if *debug {
		f.Logger = log.New(log.Stderr, fmt.Sprintf("[%s] ", conn.RemoteAddr()))
	} else {
		f.Logger = log.Discard
	}
	f.Println("face created")
	return
}
