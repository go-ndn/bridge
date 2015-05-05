package main

import (
	"fmt"

	"github.com/go-ndn/mux"
	"github.com/go-ndn/ndn"
	"github.com/go-ndn/tlv"
)

type face struct {
	ndn.Face
}

func (f *face) log(i ...interface{}) {
	if !*debug {
		return
	}
	fmt.Printf("[%s] %s", f.RemoteAddr(), fmt.Sprintln(i...))
}

func (f *face) register(name string, cost uint64) error {
	f.log("register", name)
	return ndn.SendControl(f, "rib", "register", &ndn.Parameters{
		Name:   ndn.NewName(name),
		Cost:   cost,
		Origin: 128,
	}, &key)
}

func (f *face) unregister(name string) error {
	f.log("unregister", name)
	return ndn.SendControl(f, "rib", "unregister", &ndn.Parameters{
		Name: ndn.NewName(name),
	}, &key)
}

func (f *face) fetchRoute() (rib []ndn.RIBEntry) {
	fetch := mux.NewFetcher()
	fetch.Use(mux.Assembler)
	tlv.UnmarshalByte(
		fetch.Fetch(f,
			&ndn.Interest{
				Name: ndn.NewName("/localhost/nfd/rib/list"),
				Selectors: ndn.Selectors{
					MustBeFresh: true,
				},
			}),
		&rib,
		128)
	return
}

func (f *face) ServeNDN(w ndn.Sender, i *ndn.Interest) {
	go func() {
		f.log("forward", i.Name)
		d, ok := <-f.SendInterest(i)
		if !ok {
			return
		}
		f.log("receive", d.Name)
		w.SendData(d)
	}()
}
