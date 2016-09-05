package main

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/go-ndn/log"
	"github.com/go-ndn/ndn"
)

var (
	flagConfig = flag.String("config", "bridge.json", "config path")
	flagDebug  = flag.Bool("debug", false, "enable logging")
)

var (
	key ndn.Key
)

func main() {
	flag.Parse()

	// config
	configFile, err := os.Open(*flagConfig)
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
		go local.ServeNDN(remote, i)
	}
}
