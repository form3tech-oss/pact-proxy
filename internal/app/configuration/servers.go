package configuration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/form3tech-oss/pact-proxy/internal/app/pactproxy"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"
)

var servers = map[string]*http.Server{}
var hostPaths = map[string]bool{}

func StartServer(url *url.URL, config *pactproxy.Config) error {
	rootServer, loaded := servers[url.Host]
	if !loaded {
		rootServer = newServer(url, config)
		servers[url.Host] = rootServer
		go func() {
			var err error
			if config.TLSCertFile != "" && config.TLSKeyFile != "" {
				err = rootServer.ListenAndServeTLS(config.TLSCertFile, config.TLSKeyFile)
			} else {
				err = rootServer.ListenAndServe()
			}
			if err != nil && err != http.ErrServerClosed {
				log.Error(err)
			}
		}()
		return nil
	}

	if strings.TrimLeft(url.Path, "/") == "" {
		if loaded {
			// don't allow two servers on the same address, with empty path
			return fmt.Errorf("proxy already running at %s", url.String())
		}
		return nil
	}

	// This is a new path for an existing server, so add another rewrite rule
	e := rootServer.Handler.(*echo.Echo)

	// don't allow two servers on the same address, with same path
	_, found := hostPaths[url.Path]
	if found {
		return fmt.Errorf("proxy already running at %s", url.String())
	}
	hostPaths[url.Path] = true
	addRewrite(e, url.Path)

	return nil
}

func ShutdownAllServers(ctx context.Context) {
	for addr, server := range servers {
		if err := server.Shutdown(ctx); err != nil {
			log.Error(err)
		}
		delete(servers, addr)
	}

	hostPaths = map[string]bool{}
}

func newServer(url *url.URL, config *pactproxy.Config) *http.Server {
	e := echo.New()
	e.HideBanner = true

	pactproxy.SetupRoutes(e, config)

	s := http.Server{
		Addr:    url.Host,
		Handler: e,
	}

	if config.TLSCAFile != "" {
		if config.TLSCertFile == "" || config.TLSKeyFile == "" {
			log.Fatalf("cannot run in mTLS mode without TLS cert and key")
		}

		caCertFile, err := os.ReadFile(config.TLSCAFile)
		if err != nil {
			log.Fatalf("error reading CA certificate: %v", err)
		}
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(caCertFile)
		s.TLSConfig = &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  certPool,
			MinVersion: tls.VersionTLS12,
		}
	}

	if strings.TrimLeft(url.Path, "/") != "" {
		hostPaths[url.Path] = true
		addRewrite(e, url.Path)
	}

	return &s
}

func addRewrite(e *echo.Echo, path string) {
	e.Pre(middleware.Rewrite(map[string]string{
		path + "/*": "/$1",
	}))
}
