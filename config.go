package main

var config struct {
	Local, Remote struct {
		Network, Address string
	}
	PrivateKeyPath string
	Cost           uint64
}
