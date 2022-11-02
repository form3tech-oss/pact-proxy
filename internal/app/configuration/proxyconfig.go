package configuration

import (
	"context"

	"github.com/form3tech-oss/pact-proxy/internal/app/pactproxy"
	"github.com/sethvargo/go-envconfig"
	log "github.com/sirupsen/logrus"
)

func NewFromEnv() pactproxy.Config {
	ctx := context.Background()

	var config pactproxy.Config
	err := envconfig.Process(ctx, &config)
	if err != nil {
		log.Fatal(err.Error())
	}
	return config
}

func ConfigureProxy(config pactproxy.Config) error {
	server, err := GetServer(&config.ServerAddress)
	if err != nil {
		return err
	}

	pactproxy.StartProxy(server, &config)
	return err
}
