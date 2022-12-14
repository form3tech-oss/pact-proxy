package main

import (
	"os"
	"os/signal"
	"strconv"
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

		proxyConfig := config
		proxyConfig.Target = proxy
		if err := configuration.ConfigureProxy(proxyConfig); err != nil {
			panic(err)
		}
	}

	adminPort := os.Getenv("ADMIN_PORT")
	port, err := strconv.Atoi(adminPort)
	if err != nil {
		port = 8080
	}

	adminServer := configuration.ServeAdminAPI(port)

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	if err := adminServer.Close(); err != nil {
		panic(err)
	}

	configuration.CloseAllServers()
}
