package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
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
	f, err := os.Open(*configPath)
	if err != nil {
		log(err)
		return
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&config)
	if err != nil {
		log(err)
		return
	}

	// key
	pem, err := ioutil.ReadFile(config.PrivateKeyPath)
	if err != nil {
		log(err)
		return
	}
	key.DecodePrivateKey(pem)
	log("key", key.Name)

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
