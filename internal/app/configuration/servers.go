package configuration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

var servers sync.Map

func GetServer(url *url.URL) (*http.ServeMux, error) {
	rootServer, rootHandler, loaded := loadOrStoreHandler(url.Host, &http.Server{Addr: fmt.Sprintf(url.Host), Handler: http.NewServeMux()})
	if !loaded {
		go func() {
			if err := rootServer.ListenAndServe(); err != nil {
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
		return rootHandler, nil
	}

	addr := fmt.Sprintf("%s/%s", url.Host, strings.TrimLeft(url.Path, "/"))
	_, proxyHandler, loaded := loadOrStoreHandler(addr, &http.Server{Handler: http.NewServeMux()})
	if loaded {
		return nil, fmt.Errorf("proxy already running at %s", addr)
	}

	rootHandler.Handle(url.Path+"/", http.StripPrefix(url.Path, proxyHandler))

	return proxyHandler, nil
}

func loadOrStoreHandler(addr string, defaultServer *http.Server) (*http.Server, *http.ServeMux, bool) {
	server, loaded := servers.LoadOrStore(addr, defaultServer)
	return server.(*http.Server), server.(*http.Server).Handler.(*http.ServeMux), loaded
}

func CloseAllServers() {
	servers.Range(func(key, _ interface{}) bool {
		server, loaded := servers.LoadAndDelete(key)
		if loaded {
			if err := server.(*http.Server).Close(); err != nil {
				log.Error(err)
			}
		}
		return true
	})
}
