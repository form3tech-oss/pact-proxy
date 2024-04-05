package configuration

import (
	"context"
	"net/url"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"

	"github.com/form3tech-oss/pact-proxy/internal/app/pactproxy"
)

func NewFromEnv() (pactproxy.Config, error) {
	ctx := context.Background()

	var config pactproxy.Config
	err := envconfig.ProcessWith(ctx, &envconfig.Config{
		Target:        &config,
		DefaultNoInit: true,
	})
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
