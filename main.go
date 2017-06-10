package main

import (
	"time"

	"github.com/go-ndn/log"
)

func main() {
	ctx, err := background()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("key", ctx.Key.Locator())

	ch := make(chan *tunnel, len(ctx.Tunnel))
	for _, tun := range ctx.Tunnel {
		ch <- tun
	}
	for tun := range ch {
		go func(tun *tunnel) {
			connect(ctx, tun)
			time.Sleep(tun.Advertise.Interval.Duration)
			ch <- tun
		}(tun)
	}
}
