package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/form3tech-oss/pact-proxy/internal/app/configuration"
	log "github.com/sirupsen/logrus"
)

func main() {
	config, err := configuration.NewFromEnv()
	if err != nil {
		log.WithError(err).Fatal("unable to load configuration")
	}
	for _, proxy := range config.Proxies {
		log.Infof("setting up proxy for %s", proxy.String())
		config.Target = proxy
		err := configuration.ConfigureProxy(config)
		if err != nil {
			panic(err)
		}
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	configuration.ShutdownAllServers(context.Background())
}
