package pactproxy

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/form3tech-oss/pact-proxy/internal/app/configuration"
	"github.com/pkg/errors"
)

type ProxyConfiguration struct {
	client http.Client
	url    string
}

func Configuration(url string) *ProxyConfiguration {
	return &ProxyConfiguration{
		client: http.Client{
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

	content, _ := json.Marshal(config)

	req, err := http.NewRequest("POST", strings.TrimSuffix(conf.url, "/")+"/proxies", bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	responseBody, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, errors.New(string(responseBody))
	}
	return New(serverAddress), err
}

func (conf *ProxyConfiguration) Reset() error {
	req, err := http.NewRequest("DELETE", strings.TrimSuffix(conf.url, "/")+"/proxies", nil)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.New("error resetting proxies")
	}
	return err
}
