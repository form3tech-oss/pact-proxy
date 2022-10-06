package pactproxy

import (
	"bytes"
	"io/ioutil"
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

var supportedMediaTypes = map[string]func([]byte, *url.URL) (requestDocument, error){
	mediaTypeJSON: ParseJSONRequest,
	mediaTypeText: ParsePlainTextRequest,
}

type api struct {
	target       *url.URL
	proxy        *httputil.ReverseProxy
	interactions *Interactions
	notify       *notify
	delay        time.Duration
	duration     time.Duration
	echo.Context
}

func (a *api) ProxyRequest(c echo.Context) error {
	a.proxy.ServeHTTP(c.Response(), c.Request())
	return nil
}

func StartProxy(e *echo.Echo, target *url.URL) {

	// Create these once at startup, thay are shared by all server threads
	a := api{
		target:       target,
		proxy:        httputil.NewSingleHostReverseProxy(target),
		interactions: &Interactions{},
		notify:       NewNotify(),
		delay:        defaultDelay,
		duration:     defaultDuration,
	}

	e.GET("/ready", a.readinessHandler)

	e.Any("/interactions/verification", a.proxyPassHandler)
	e.Any("/pact", a.proxyPassHandler)

	e.POST("/interactions/constraints", a.interactionsConstraintsHandler) // TODO is this a POST ?
	e.POST("/interactions/modifiers", a.interactionsModifiersHandler)     // TODO is this a POST ?

	e.DELETE("/session", a.sessionHandler)

	e.POST("/interactions", a.interactionsPostHandler)
	e.DELETE("/interactions", a.interactionsDeleteHandler)

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
	interaction.Modifiers.AddModifier(modifier)

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
	data, err := ioutil.ReadAll(c.Request().Body)
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

	a.interactions.Store(interaction)

	err = c.Request().Body.Close()
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Error(err.Error()))
	}

	c.Request().Body = ioutil.NopCloser(bytes.NewBuffer(data))

	return a.ProxyRequest(c)
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

		log.Infof("waiting for %s", waitFor)
		retryFor(func(timeLeft time.Duration) bool {
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

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to read requestDocument data. %s", err.Error()))
	}

	err = c.Request().Body.Close()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, httpresponse.Error(err.Error()))
	}

	c.Request().Body = ioutil.NopCloser(bytes.NewBuffer(data))

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
	matched := make([]*interaction, 0)
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
