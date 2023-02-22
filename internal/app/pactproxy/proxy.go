package pactproxy

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/form3tech-oss/pact-proxy/internal/app/httpresponse"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

const (
	defaultDelay    = 500 * time.Millisecond
	defaultDuration = 15 * time.Second
)

type Config struct {
	ServerAddress url.URL       `env:"SERVER_ADDRESS"`      // Address to listen on
	Proxies       []url.URL     `env:"PROXIES,delimiter=;"` // List of URL to serve pact-proxy on, e.g. http://localhost:8080;http://localhost:8081
	WaitDelay     time.Duration `env:"WAIT_DELAY"`          // Default Delay for WaitForInteractions endpoint
	WaitDuration  time.Duration `env:"WAIT_DURATION"`       // Default Duration for WaitForInteractions endpoint
	RecordHistory bool          `env:"RECORD_HISTORY"`
	TLSCAFile     string        `env:"TLS_CA_FILE"`
	TLSCertFile   string        `env:"TLS_CERT_FILE"`
	TLSKeyFile    string        `env:"TLS_KEY_FILE"`
	Target        url.URL       // Do not load Target from env, we set this for each value from Proxies
}

var supportedMediaTypes = map[string]func([]byte, *url.URL) (requestDocument, error){
	mediaTypeJSON: ParseJSONRequest,
	mediaTypeText: ParsePlainTextRequest,
}

type api struct {
	target        *url.URL
	proxy         *httputil.ReverseProxy
	interactions  *Interactions
	notify        *notify
	delay         time.Duration
	duration      time.Duration
	recordHistory bool
	echo.Context
}

func (a *api) ProxyRequest(c echo.Context) error {
	a.proxy.ServeHTTP(c.Response(), c.Request())
	return nil
}

func SetupRoutes(e *echo.Echo, config *Config) {
	// Create these once at startup, thay are shared by all server threads
	a := api{
		target:        &config.Target,
		proxy:         httputil.NewSingleHostReverseProxy(&config.Target),
		interactions:  &Interactions{},
		notify:        NewNotify(),
		delay:         config.WaitDelay,
		duration:      config.WaitDuration,
		recordHistory: config.RecordHistory,
	}
	if a.delay == 0 {
		a.delay = defaultDelay
	}
	if a.duration == 0 {
		a.duration = defaultDuration
	}

	e.GET("/ready", a.readinessHandler)

	e.Any("/interactions/verification", a.proxyPassHandler)
	e.Any("/pact", a.proxyPassHandler)

	e.POST("/interactions/constraints", a.interactionsConstraintsHandler)
	e.POST("/interactions/modifiers", a.interactionsModifiersHandler)

	e.DELETE("/session", a.sessionHandler)

	e.POST("/interactions", a.interactionsPostHandler)
	e.DELETE("/interactions", a.interactionsDeleteHandler)

	e.GET("/interactions/details/:alias", a.interactionsGetHandler)
	e.GET("/interactions/wait", a.interactionsWaitHandler)

	e.Any("/*", a.indexHandler)
}

func (a *api) proxyPassHandler(c echo.Context) error {
	return a.ProxyRequest(c)
}

func (a *api) readinessHandler(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

func (a *api) interactionsConstraintsHandler(c echo.Context) error {
	constraint := interactionConstraint{}
	err := c.Bind(&constraint)
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to load constraint. %s", err.Error()))
	}

	interaction, ok := a.interactions.Load(constraint.Interaction)
	if !ok {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to find interaction. %s", constraint.Interaction))
	}

	log.Infof("adding constraint to interaction '%s'", interaction.Description)
	interaction.AddConstraint(constraint)

	return c.NoContent(http.StatusOK)
}

func (a *api) interactionsModifiersHandler(c echo.Context) error {
	modifier := &interactionModifier{}
	err := c.Bind(modifier)
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to load modifier. %s", err.Error()))
	}

	interaction, ok := a.interactions.Load(modifier.Interaction)
	if !ok {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to find interaction for modifier. %s", modifier.Interaction))
	}

	log.Infof("adding modifier to interaction '%s'", interaction.Description)
	interaction.modifiers.AddModifier(modifier)

	return c.NoContent(http.StatusOK)
}

func (a *api) sessionHandler(c echo.Context) error {
	log.Infof("deleting session for %s", a.target)
	return a.ProxyRequest(c)
}

func (a *api) interactionsDeleteHandler(c echo.Context) error {
	log.Info("deleting interactions")
	a.ProxyRequest(c)
	a.interactions.Clear()
	return nil
}

func (a *api) interactionsPostHandler(c echo.Context) error {
	data, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to read interaction. %s", err.Error()))
	}

	interaction, err := LoadInteraction(data, c.QueryParam("alias"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to load interaction. %s", err.Error()))
	}

	if interaction.Alias != "" {
		log.Infof("storing interaction '%s' (%s)", interaction.Description, interaction.Alias)
	} else {
		log.Infof("storing interaction '%s'", interaction.Description)
	}

	interaction.recordHistory = a.recordHistory
	a.interactions.Store(interaction)

	err = c.Request().Body.Close()
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Error(err.Error()))
	}

	c.Request().Body = io.NopCloser(bytes.NewBuffer(data))

	return a.ProxyRequest(c)
}

func (a *api) interactionsGetHandler(c echo.Context) error {
	alias := c.Param("alias")
	interaction, found := a.interactions.Load(alias)
	if !found {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("interaction %q not found", alias))
	}

	return c.JSON(http.StatusOK, interaction)
}

func (a *api) interactionsWaitHandler(c echo.Context) error {
	waitForCount, err := strconv.Atoi(c.QueryParam("count"))
	if err != nil {
		waitForCount = 1
	}

	if waitFor := c.QueryParam("interaction"); waitFor != "" {
		interaction, ok := a.interactions.Load(waitFor)
		if !ok {
			return c.JSON(http.StatusBadRequest, httpresponse.Errorf("cannot wait for interaction '%s', interaction not found.", waitFor))
		}

		log.WithField("wait_for", waitFor).Infof("waiting")
		retryFor(func(timeLeft time.Duration) bool {
			log.WithFields(log.Fields{
				"wait_for":       waitFor,
				"count":          waitForCount,
				"time_remaining": timeLeft,
			}).Infof("retry")
			if interaction.HasRequests(waitForCount) {
				return true
			}
			if timeLeft > 0 {
				a.notify.Wait(timeLeft)
			}
			return false
		}, a.delay, a.duration)

		if !interaction.HasRequests(waitForCount) {
			return c.JSON(http.StatusRequestTimeout, httpresponse.Error("timeout waiting for interactions to be met"))
		}

		return c.NoContent(http.StatusOK)
	}

	log.Info("waiting for all")
	retryFor(func(timeLeft time.Duration) bool {
		if a.interactions.AllHaveRequests() {
			return true
		}
		if timeLeft > 0 {
			a.notify.Wait(timeLeft)
		}
		return false
	}, a.delay, a.duration)

	if !a.interactions.AllHaveRequests() {
		for _, i := range a.interactions.All() {
			if !i.HasRequests(1) {
				log.Infof("'%s' has no requests", i.Description)
			}
		}

		return c.JSON(http.StatusRequestTimeout, httpresponse.Error("timeout waiting for interactions to be met"))
	}
	return c.NoContent(http.StatusOK)
}

func (a *api) indexHandler(c echo.Context) error {
	req := c.Request()
	log.Infof("proxying %s %s", req.Method, req.URL.Path)

	mediaType, err := parseMediaTypeHeader(c.Request().Header)
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("failed to parse Content-Type header. %s", err.Error()))
	}

	parseRequest, ok := supportedMediaTypes[mediaType]
	if !ok {
		return c.JSON(http.StatusUnsupportedMediaType, httpresponse.Errorf("unsupported Media Type: %s", mediaType))
	}

	allInteractions, ok := a.interactions.FindAll(req.URL.Path, req.Method)
	if !ok {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to find interaction to Match '%s %s'", req.Method, req.URL.Path))
	}

	data, err := io.ReadAll(req.Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to read requestDocument data. %s", err.Error()))
	}

	err = c.Request().Body.Close()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, httpresponse.Error(err.Error()))
	}

	c.Request().Body = io.NopCloser(bytes.NewBuffer(data))

	request, err := parseRequest(data, c.Request().URL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, httpresponse.Errorf("unable to read requestDocument data. %s", err.Error()))
	}
	h := make(map[string]interface{})
	for headerName, headerValues := range c.Request().Header {
		for _, headerValue := range headerValues {
			h[headerName] = headerValue
		}
	}
	request["headers"] = h

	unmatched := make(map[string][]string)
	matched := make([]*Interaction, 0)
	for _, interaction := range allInteractions {
		ok, info := interaction.EvaluateConstrains(request, a.interactions)
		if ok {
			interaction.StoreRequest(request)
			matched = append(matched, interaction)
		} else {
			unmatched[interaction.Description] = info
		}
	}

	if len(unmatched) == len(allInteractions) {
		for desc, info := range unmatched {
			results := strings.Join(info, "\n")
			log.Infof("constraints do not match for '%s'.\n\n%s", desc, results)
		}
		return c.JSON(http.StatusBadRequest, httpresponse.Error("constraints do not match"))
	}

	a.notify.Notify()
	a.proxy.ServeHTTP(&ResponseModificationWriter{res: c.Response(), interactions: matched}, req)
	return nil
}

func parseMediaTypeHeader(header http.Header) (string, error) {
	contentType := header.Get("Content-Type")
	if contentType == "" {
		log.Info("Request does not have Content-Type header - defaulting to text/plain")
		return mediaTypeText, nil
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", err
	}
	return mediaType, nil
}
