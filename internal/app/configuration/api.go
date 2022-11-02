package configuration

import (
	"fmt"
	"net/http"

	"github.com/form3tech-oss/pact-proxy/internal/app/httpresponse"
	"github.com/form3tech-oss/pact-proxy/internal/app/pactproxy"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

func ServeAdminAPI(port int) *echo.Echo {
	adminServer := echo.New()
	adminServer.HideBanner = true

	adminServer.DELETE("/proxies", deleteProxiesHandler)
	adminServer.POST("/proxies", postProxiesHandler)

	go func() {
		address := fmt.Sprintf(":%d", port)
		if err := adminServer.Start(address); err != nil {
			log.Fatal(err)
		}
	}()

	return adminServer
}

func deleteProxiesHandler(c echo.Context) error {
	log.Infof("closing all proxies")
	CloseAllServers()
	return c.NoContent(http.StatusNoContent)
}

func postProxiesHandler(c echo.Context) error {

	proxyConfig := pactproxy.Config{}
	err := c.Bind(&proxyConfig)
	if err != nil {
		return c.JSON(
			http.StatusBadRequest,
			httpresponse.Errorf("unable to parse interactionConstraint from data. %s", err.Error()),
		)
	}

	log.Infof("setting up proxy from %s to %s", proxyConfig.ServerAddress, proxyConfig.Target)

	err = ConfigureProxy(proxyConfig)
	if err != nil {
		return c.JSON(
			http.StatusInternalServerError,
			httpresponse.Errorf("unable to create proxy from configuration. %s", err.Error()),
		)
	}

	return c.NoContent(http.StatusNoContent)
}
