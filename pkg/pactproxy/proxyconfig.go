package pactproxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/form3tech-oss/pact-proxy/internal/app/configuration"
	"github.com/pkg/errors"
)

var (
	ErrProxyConfigResseting = errors.New("error resetting proxies")
)

type ProxyConfiguration struct {
	client *http.Client
	url    string
}

func Configuration(url string) *ProxyConfiguration {
	return &ProxyConfiguration{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		url: url,
	}
}

func (conf *ProxyConfiguration) SetupProxy(serverAddress, targetAddress string) (*PactProxy, error) {
	config := &configuration.ProxyConfig{
		ServerAddress: serverAddress,
		Target:        targetAddress,
	}

	content, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/proxies", conf.url)
	req, err := http.NewRequest("POST", url, bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := conf.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	responseBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("bad status: %s", responseBody)
	}

	return New(serverAddress), err
}

func (conf *ProxyConfiguration) Reset() error {
	url := fmt.Sprintf("%s/proxies", conf.url)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}

	res, err := conf.client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return ErrProxyConfigResseting
	}

	return nil
}
