package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"

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

	go advertise(local, remote)

	// create remote tunnel
	for i := range recv {
		local.ServeNDN(remote, i)
	}
}

func advertise(local, remote *face) {
	// true = fresh, false = stale
	registered := make(map[string]bool)
	for {
		localRoutes := local.fetchRoute()
		remoteRoutes := remote.fetchRoute()
		index := make(map[string]uint64)
		for _, routes := range remoteRoutes {
			name := routes.Name.String()
			for _, route := range routes.Route {
				if cost, ok := index[name]; ok && cost <= route.Cost {
					continue
				}
				index[name] = route.Cost
			}
		}
		for _, routes := range localRoutes {
			name := routes.Name.String()
			for _, route := range routes.Route {
				advCost := route.Cost + config.Cost
				if cost, ok := index[name]; ok && cost < advCost {
					continue
				}
				if _, ok := registered[name]; !ok {
					err := remote.register(name, advCost)
					if err != nil {
						remote.log(err)
					}
				}
				registered[name] = true
				break
			}
		}
		for name, fresh := range registered {
			if fresh {
				registered[name] = false
			} else {
				delete(registered, name)
				err := remote.unregister(name)
				if err != nil {
					remote.log(err)
				}
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func log(i ...interface{}) {
	if !*debug {
		return
	}
	fmt.Printf("[bridge] %s", fmt.Sprintln(i...))
}
