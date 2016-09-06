package main

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/go-ndn/ndn"
)

type context struct {
	Local, Remote struct {
		Network, Address string
	}
	PrivateKeyPath string
	Cost           uint64
	Debug          bool    `json:"-"`
	ConfigPath     string  `json:"-"`
	Key            ndn.Key `json:"-"`
}

func background() (*context, error) {
	var ctx context
	flag.StringVar(&ctx.ConfigPath, "config", "bridge.json", "config path")
	flag.BoolVar(&ctx.Debug, "debug", false, "enable logging")

	flag.Parse()

	configFile, err := os.Open(ctx.ConfigPath)
	if err != nil {
		return nil, err
	}
	defer configFile.Close()

	err = json.NewDecoder(configFile).Decode(&ctx)
	if err != nil {
		return nil, err
	}

	// key
	pem, err := os.Open(ctx.PrivateKeyPath)
	if err != nil {
		return nil, err
	}
	defer pem.Close()

	ctx.Key, err = ndn.DecodePrivateKey(pem)
	if err != nil {
		return nil, err
	}
	return &ctx, nil
}
