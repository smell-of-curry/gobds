package main

import (
	"log/slog"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/smell-of-curry/gobds/gobds"
)

func main() {
	log := slog.Default()
	conf, err := gobds.ReadConfig()
	if err != nil {
		panic(err)
	}

	g := gobds.NewGoBDS(conf, log)
	err = retry.Do(
		g.Start,
		retry.Attempts(5),
		retry.Delay(time.Second*3),
		retry.OnRetry(func(n uint, err error) {
			log.Error("failed to start, retrying", "attempt", n+1, "error", err)
		}),
	)
	if err != nil {
		log.Error("failed to start after multiple retries, shutting down", "error", err)
		return
	}
}
