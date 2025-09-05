package main

import (
	"log/slog"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/getsentry/sentry-go"
	"github.com/smell-of-curry/gobds/gobds"
)

func main() {
	log := slog.Default()
	conf, err := gobds.ReadConfig()
	if err != nil {
		panic(err)
	}

	dsn := conf.Network.SentryDSN
	if dsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:        conf.Network.SentryDSN,
			ServerName: conf.Network.ServerName,
		})
		if err != nil {
			panic(err)
		}
		defer sentry.Flush(2 * time.Second)
	}

	c, err := conf.Config(slog.Default())
	if err != nil {
		panic(err)
	}

	g := c.New()
	g.CloseOnProgramEnd()
	err = retry.Do(
		g.Listen,
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
