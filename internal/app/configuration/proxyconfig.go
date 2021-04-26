package configuration

import (
	"fmt"
	"net/url"

	"github.com/form3tech-oss/pact-proxy/internal/app/pactproxy"
)

type ProxyConfig struct {
	ServerAddress string
	Target        string
}

func ConfigureProxy(config ProxyConfig) error {
	targetURL, err := url.Parse(config.Target)
	if err != nil {
		return err
	}

	serverAddr := config.ServerAddress
	if serverAddr == "" {
		serverAddr = fmt.Sprintf("http://:%s", targetURL.Port())
	}

	serverAddrURL, err := url.Parse(serverAddr)
	if err != nil {
		return err
	}

	server, err := GetServer(serverAddrURL)
	if err != nil {
		return err
	}

	pactproxy.StartProxy(server, targetURL)
	return err
}
