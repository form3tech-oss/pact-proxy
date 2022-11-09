package configuration

import (
	"context"

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
	server, err := GetServer(&config.ServerAddress)
	if err != nil {
		return err
	}

	pactproxy.StartProxy(server, &config)
	return err
}
