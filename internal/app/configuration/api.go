package configuration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/form3tech-oss/pact-proxy/internal/app/httpresponse"
	log "github.com/sirupsen/logrus"
)

func ServeAdminAPI(port int) *http.Server {
	adminServerHandler := http.NewServeMux()
	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: adminServerHandler,
	}

	adminServerHandler.HandleFunc("/proxies", adminHandler)

	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	return s
}

func adminHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodDelete {
		log.Infof("closing all proxies")
		CloseAllServers()
		return
	}

	if req.Method == http.MethodPost {
		configBytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			httpresponse.Errorf(w, http.StatusBadRequest, "unable to read constraint. %s", err.Error())
			return
		}

		proxyConfig := ProxyConfig{}
		err = json.Unmarshal(configBytes, &proxyConfig)
		if err != nil {
			httpresponse.Errorf(w, http.StatusBadRequest, "unable to parse interactionConstraint from data. %s", err.Error())
			return
		}

		log.Infof("setting up proxy from %s to %s", proxyConfig.ServerAddress, proxyConfig.Target)

		err = ConfigureProxy(proxyConfig)
		if err != nil {
			httpresponse.Errorf(w, http.StatusInternalServerError, "unable to create proxy from configuration. %s", err.Error())
			return
		}
	}
}
