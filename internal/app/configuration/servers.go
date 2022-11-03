package configuration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

var servers sync.Map

func GetServer(url *url.URL) (*echo.Echo, error) {
	rootServer, loaded := loadOrStoreHandler(url.Host, echo.New())
	if !loaded {
		go func() {
			rootServer.HideBanner = true
			if err := rootServer.Start(url.Host); err != nil {
				if err != http.ErrServerClosed {
					log.Error(err)
				}
			}
		}()
	}

	if strings.TrimLeft(url.Path, "/") == "" {
		if loaded {
			return nil, fmt.Errorf("proxy already running at %s", url.String())
		}
		return rootServer, nil
	}

	addr := fmt.Sprintf("%s/%s", url.Host, strings.TrimLeft(url.Path, "/"))
	proxyServer, loaded := loadOrStoreHandler(addr, echo.New())
	if loaded {
		return nil, fmt.Errorf("proxy already running at %s", addr)
	}
	proxyServer.HideBanner = true

	rootServer.Any(url.Path+"/", echo.WrapHandler(http.StripPrefix(url.Path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyServer.ServeHTTP(w, r)
	}))))

	return proxyServer, nil
}

func loadOrStoreHandler(addr string, defaultServer *echo.Echo) (*echo.Echo, bool) {
	server, loaded := servers.LoadOrStore(addr, defaultServer)
	return server.(*echo.Echo), loaded
}

func CloseAllServers() {
	servers.Range(func(key, _ interface{}) bool {
		server, loaded := servers.LoadAndDelete(key)
		if loaded {
			if err := server.(*echo.Echo).Close(); err != nil {
				log.Error(err)
			}
		}
		return true
	})
}
