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

func newFace(ctx *context, network, address string, recv chan<- *ndn.Interest) (*face, error) {
	conn, err := packet.Dial(network, address)
	if err != nil {
		return nil, err
	}
	f := &face{
		Face:    ndn.NewFace(conn, recv),
		Fetcher: mux.NewFetcher(),
	}
	f.Fetcher.Use(mux.Assembler)

	if ctx.Debug {
		f.Logger = log.New(log.Stderr, fmt.Sprintf("[%s] ", conn.RemoteAddr()))
	} else {
		f.Logger = log.Discard
	}
	f.Println("face created")
	return f, nil
}

func (f *face) register(ctx *context, name string, cost uint64) error {
	f.Println("register", name)
	return ndn.SendControl(f, "rib", "register", &ndn.Parameters{
		Name:   ndn.NewName(name),
		Cost:   cost,
		Origin: 128,
	}, ctx.Key)
}

func (f *face) unregister(ctx *context, name string) error {
	f.Println("unregister", name)
	return ndn.SendControl(f, "rib", "unregister", &ndn.Parameters{
		Name: ndn.NewName(name),
	}, ctx.Key)
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

func connect(ctx *context, tun *tunnel) {
	// local face
	recvLocal := make(chan *ndn.Interest)
	local, err := newFace(ctx, tun.Local.Network, tun.Local.Address, recvLocal)
	if err != nil {
		log.Println(err)
		return
	}
	defer local.Close()
	// remote face
	recvRemote := make(chan *ndn.Interest)
	remote, err := newFace(ctx, tun.Remote.Network, tun.Remote.Address, recvRemote)
	if err != nil {
		log.Println(err)
		return
	}
	defer remote.Close()

	done := make(chan struct{})
	defer close(done)

	go advertise(ctx, &advertiseOptions{
		Remote:   remote,
		Local:    local,
		Cost:     tun.Advertise.Cost,
		Interval: tun.Advertise.Interval.Duration,
		Done:     done,
	})

	if tun.Undirected {
		go advertise(ctx, &advertiseOptions{
			Remote:   local,
			Local:    remote,
			Cost:     tun.Advertise.Cost,
			Interval: tun.Advertise.Interval.Duration,
			Done:     done,
		})
	}

	// create remote tunnel
	for {
		select {
		case i, ok := <-recvLocal:
			if !ok {
				return
			}
			go remote.ServeNDN(local, i)
		case i, ok := <-recvRemote:
			if !ok {
				return
			}
			go local.ServeNDN(remote, i)
		}
	}
}

type advertiseOptions struct {
	Remote, Local *face
	Cost          uint64
	Interval      time.Duration
	Done          <-chan struct{}
}

func advertise(ctx *context, opt *advertiseOptions) {
	// true = fresh, false = stale
	registered := make(map[string]bool)
	for {
		select {
		case <-opt.Done:
			return
		case <-time.After(opt.Interval):
			localRoutes := opt.Local.fetchRoute()
			remoteRoutes := opt.Remote.fetchRoute()
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
					advCost := route.Cost + opt.Cost
					if cost, ok := index[name]; ok && cost < advCost {
						continue
					}
					if _, ok := registered[name]; !ok {
						err := opt.Remote.register(ctx, name, advCost)
						if err != nil {
							opt.Remote.Println(err)
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
					err := opt.Remote.unregister(ctx, name)
					if err != nil {
						opt.Remote.Println(err)
					}
				}
			}
		}
	}
}

func (f *face) ServeNDN(w ndn.Sender, i *ndn.Interest) {
	f.Println("forward", i.Name)
	d, err := f.SendInterest(i)
	if err != nil {
		return
	}
	f.Println("receive", d.Name)
	w.SendData(d)
}
