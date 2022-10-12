package pactproxy

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/form3tech-oss/pact-proxy/internal/app/pactproxy"
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
	serverURL, err := url.Parse(serverAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse server address")
	}
	targetURL, err := url.Parse(targetAddress)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse target address")
	}

	config := &pactproxy.Config{
		ServerAddress: *serverURL,
		Target:        *targetURL,
	}

	content, err := json.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal config")
	}

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
