package main

import (
	"fmt"
	"time"

	"github.com/go-ndn/log"
	"github.com/go-ndn/mux"
	"github.com/go-ndn/ndn"
	"github.com/go-ndn/packet"
	"github.com/go-ndn/tlv"
)

type face struct {
	ndn.Face
	*mux.Fetcher
	log.Logger
}

func newFace(network, address string, recv chan<- *ndn.Interest) (f *face, err error) {
	conn, err := packet.Dial(network, address)
	if err != nil {
		return
	}
	f = &face{
		Face:    ndn.NewFace(conn, recv),
		Fetcher: mux.NewFetcher(),
	}
	f.Fetcher.Use(mux.Assembler)

	if *flagDebug {
		f.Logger = log.New(log.Stderr, fmt.Sprintf("[%s] ", conn.RemoteAddr()))
	} else {
		f.Logger = log.Discard
	}
	f.Println("face created")
	return
}

func (f *face) register(name string, cost uint64) error {
	f.Println("register", name)
	return ndn.SendControl(f, "rib", "register", &ndn.Parameters{
		Name:   ndn.NewName(name),
		Cost:   cost,
		Origin: 128,
	}, key)
}

func (f *face) unregister(name string) error {
	f.Println("unregister", name)
	return ndn.SendControl(f, "rib", "unregister", &ndn.Parameters{
		Name: ndn.NewName(name),
	}, key)
}

func (f *face) fetchRoute() (rib []ndn.RIBEntry) {
	tlv.Unmarshal(
		f.Fetch(f,
			&ndn.Interest{
				Name: ndn.NewName("/localhop/nfd/rib/list"),
				Selectors: ndn.Selectors{
					MustBeFresh: true,
				},
			}),
		&rib,
		128,
	)
	return
}

const (
	advertiseIntv = 5 * time.Second
)

func (f *face) advertise(remote *face) {
	// true = fresh, false = stale
	registered := make(map[string]bool)
	for {
		localRoutes := f.fetchRoute()
		remoteRoutes := remote.fetchRoute()
		// for each name, find the best remote route.
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
		// if any local route is not worse, mark the name as fresh.
		// if the name is not registered, register it to remote.
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
						remote.Println(err)
					}
				}
				registered[name] = true
				break
			}
		}
		// sweep registered names.
		// if the name is fresh, mark it as stale for the next iteration.
		// otherwise, unregister, and clean up.
		for name, fresh := range registered {
			if fresh {
				registered[name] = false
			} else {
				delete(registered, name)
				err := remote.unregister(name)
				if err != nil {
					remote.Println(err)
				}
			}
		}

		time.Sleep(advertiseIntv)
	}
}

func (f *face) ServeNDN(w ndn.Sender, i *ndn.Interest) {
	f.Println("forward", i.Name)
	d, ok := <-f.SendInterest(i)
	if !ok {
		return
	}
	f.Println("receive", d.Name)
	w.SendData(d)
}
