package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/go-ndn/ndn"
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
		log(err)
		return
	}
	defer configFile.Close()

	err = json.NewDecoder(configFile).Decode(&config)
	if err != nil {
		log(err)
		return
	}

	// key
	pem, err := os.Open(config.PrivateKeyPath)
	if err != nil {
		log(err)
		return
	}
	defer pem.Close()

	key, err = ndn.DecodePrivateKey(pem)
	if err != nil {
		log(err)
		return
	}
	log("key", key.Locator())

	// local face
	conn, err := net.Dial(config.Local.Network, config.Local.Address)
	if err != nil {
		log(err)
		return
	}
	local := &face{ndn.NewFace(conn, nil)}
	defer local.Close()
	// remote face
	conn, err = net.Dial(config.Remote.Network, config.Remote.Address)
	if err != nil {
		log(err)
		return
	}
	recv := make(chan *ndn.Interest)
	remote := &face{ndn.NewFace(conn, recv)}
	defer remote.Close()

	go local.advertise(remote)

	// create remote tunnel
	for i := range recv {
		local.ServeNDN(remote, i)
	}
}

func log(i ...interface{}) {
	if !*debug {
		return
	}
	fmt.Printf("[bridge] %s", fmt.Sprintln(i...))
}
