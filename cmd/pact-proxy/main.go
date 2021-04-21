package main

import (
	"strconv"

	"github.com/form3tech/pact-proxy/internal/app/configuration"
	log "github.com/sirupsen/logrus"

	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	proxies := os.Getenv("PROXIES")
	for _, proxy := range strings.Split(strings.TrimSpace(proxies), ";") {
		if proxy != "" {
			log.Infof("setting up proxy for %s", proxy)
			if err := configuration.ConfigureProxy(configuration.ProxyConfig{Target: proxy}); err != nil {
				panic(err)
			}
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
