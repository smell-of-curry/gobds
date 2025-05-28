package main

import (
	"log/slog"

	"github.com/smell-of-curry/gobds/gobds"
)

func main() {
	log := slog.Default()
	conf, err := gobds.ReadConfig()
	if err != nil {
		panic(err)
	}

	g := gobds.NewGoBDS(conf, log)
	if err = g.Start(); err != nil {
		panic(err)
	}
}
