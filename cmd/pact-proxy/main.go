package main

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/form3tech-oss/pact-proxy/internal/app/configuration"
	"github.com/form3tech-oss/pact-proxy/internal/app/pactproxy"
	log "github.com/sirupsen/logrus"
)

func main() {
	config := configuration.NewFromEnv()
	for _, proxy := range config.Proxies {
		log.Infof("setting up proxy for %s", proxy)
		if err := configuration.ConfigureProxy(pactproxy.Config{Target: proxy}); err != nil {
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
