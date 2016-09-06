package main

import (
	"time"

	"github.com/go-ndn/log"
	"github.com/go-ndn/ndn"
)

func main() {
	ctx, err := background()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("key", ctx.Key.Locator())

	// local face
	local, err := newFace(ctx, ctx.Local.Network, ctx.Local.Address, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer local.Close()
	// remote face
	recv := make(chan *ndn.Interest)
	remote, err := newFace(ctx, ctx.Remote.Network, ctx.Remote.Address, recv)
	if err != nil {
		log.Println(err)
		return
	}
	defer remote.Close()

	go advertise(ctx, remote, local, 5*time.Second)

	// create remote tunnel
	for i := range recv {
		go local.ServeNDN(remote, i)
	}
}
