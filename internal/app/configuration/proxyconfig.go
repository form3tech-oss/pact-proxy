package configuration

import (
	"context"
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
	targetURL := config.Target

	// If ServerAddress is not passed, listen on the target port
	serverAddr := config.ServerAddress
	if (serverAddr == url.URL{}) {
		serverAddr = url.URL{Scheme: "http", Host: ":" + targetURL.Port()}
	}

	return StartServer(&serverAddr, &config)
}
