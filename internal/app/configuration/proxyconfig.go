package configuration

import (
	"context"
	"fmt"
	"net/url"

	"github.com/form3tech-oss/pact-proxy/internal/app/pactproxy"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
)

func NewFromEnv() (pactproxy.Config, error) {
	ctx := context.Background()

	var config pactproxy.Config
	err := envconfig.Process(ctx, &config)
	if err != nil {
		return config, errors.Wrap(err, "process env config")
	}
	return config, nil
}

func ConfigureProxy(config pactproxy.Config) error {
	serverAddr, err := getServerAddr(config)
	if err != nil {
		return err
	}

	server, err := GetServer(serverAddr)
	if err != nil {
		return err
	}

	pactproxy.StartProxy(server, &config)
	return err
}

func getServerAddr(config pactproxy.Config) (*url.URL, error) {
	if config.ServerAddress.Host != "" {
		return &config.ServerAddress, nil
	}
	return url.Parse(fmt.Sprintf("http://:%s", config.Target.Port()))
}
