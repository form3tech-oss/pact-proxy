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

type ProxyContext struct {
	target       *url.URL
	proxy        *httputil.ReverseProxy
	interactions *Interactions
	notify       *notify
	delay        time.Duration
	duration     time.Duration
	echo.Context
}

func (cc *ProxyContext) ProxyRequest() error {
	cc.proxy.ServeHTTP(cc.Response(), cc.Request())
	return nil
}

func StartProxy(e *echo.Echo, target *url.URL) {

	// Create these once at startup, thay are shared by all server threads
	proxy := httputil.NewSingleHostReverseProxy(target)
	notify := NewNotify()
	interactions := &Interactions{}

	// Create a middleware to extend default context, adding the api params
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &ProxyContext{
				target:       target,
				proxy:        proxy,
				interactions: interactions,
				notify:       notify,
				delay:        defaultDelay,
				duration:     defaultDuration,
				Context:      c,
			}
			return next(cc)
		}
	})

	e.GET("/ready", readinessHandler)

	e.Any("/interactions/verification", proxyPassHandler)
	e.Any("/pact", proxyPassHandler)

	e.POST("/interactions/constraints", interactionsConstraintsHandler) // TODO is this a POST ?
	e.POST("/interactions/modifiers", interactionsModifiersHandler)     // TODO is this a POST ?

	e.DELETE("/session", sessionHandler)

	e.POST("/interactions", interactionsPostHandler)
	e.DELETE("/interactions", interactionsDeleteHandler)

	e.GET("/interactions/wait", interactionsWaitHandler)

	e.Any("/*", indexHandler)
}

func proxyPassHandler(c echo.Context) error {
	cc := c.(*ProxyContext)
	return cc.ProxyRequest()
}

func readinessHandler(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

func interactionsConstraintsHandler(c echo.Context) error {
	cc := c.(*ProxyContext)

	constraint := interactionConstraint{}
	err := cc.Bind(&constraint)
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to load constraint. %s", err.Error()))
	}

	interaction, ok := cc.interactions.Load(constraint.Interaction)
	if !ok {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to find interaction. %s", constraint.Interaction))
	}

	log.Infof("adding constraint to interaction '%s'", interaction.Description)
	interaction.AddConstraint(constraint)

	return c.NoContent(http.StatusOK)
}

func interactionsModifiersHandler(c echo.Context) error {
	cc := c.(*ProxyContext)
	modifier := &interactionModifier{}
	err := cc.Bind(modifier)
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to load modifier. %s", err.Error()))
	}

	interaction, ok := cc.interactions.Load(modifier.Interaction)
	if !ok {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to find interaction for modifier. %s", modifier.Interaction))
	}

	log.Infof("adding modifier to interaction '%s'", interaction.Description)
	interaction.Modifiers.AddModifier(modifier)

	return c.NoContent(http.StatusOK)
}

func sessionHandler(c echo.Context) error {
	cc := c.(*ProxyContext)
	log.Infof("deleting session for %s", cc.target)
	cc.ProxyRequest()
	return nil
}

func interactionsDeleteHandler(c echo.Context) error {
	cc := c.(*ProxyContext)
	log.Info("deleting interactions")
	cc.ProxyRequest()
	cc.interactions.Clear()
	return nil
}

func interactionsPostHandler(c echo.Context) error {
	cc := c.(*ProxyContext)
	data, err := ioutil.ReadAll(cc.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to read interaction. %s", err.Error()))
	}

	interaction, err := LoadInteraction(data, cc.QueryParam("alias"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to load interaction. %s", err.Error()))
	}

	if interaction.Alias != "" {
		log.Infof("storing interaction '%s' (%s)", interaction.Description, interaction.Alias)
	} else {
		log.Infof("storing interaction '%s'", interaction.Description)
	}

	cc.interactions.Store(interaction)

	err = cc.Request().Body.Close()
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Error(err.Error()))
	}

	cc.Request().Body = ioutil.NopCloser(bytes.NewBuffer(data))

	return cc.ProxyRequest()
}

func interactionsWaitHandler(c echo.Context) error {
	cc := c.(*ProxyContext)
	waitForCount, err := strconv.Atoi(cc.QueryParam("count"))
	if err != nil {
		waitForCount = 1
	}

	if waitFor := cc.QueryParam("interaction"); waitFor != "" {
		interaction, ok := cc.interactions.Load(waitFor)
		if !ok {
			return c.JSON(http.StatusBadRequest, httpresponse.Errorf("cannot wait for interaction '%s', interaction not found.", waitFor))
		}

		log.Infof("waiting for %s", waitFor)
		retryFor(func(timeLeft time.Duration) bool {
			if interaction.HasRequests(waitForCount) {
				return true
			}
			if timeLeft > 0 {
				cc.notify.Wait(timeLeft)
			}
			return false
		}, cc.delay, cc.duration)

		if !interaction.HasRequests(waitForCount) {
			return c.JSON(http.StatusRequestTimeout, httpresponse.Error("timeout waiting for interactions to be met"))
		}

		return c.NoContent(http.StatusOK)
	}

	log.Info("waiting for all")
	retryFor(func(timeLeft time.Duration) bool {
		if cc.interactions.AllHaveRequests() {
			return true
		}
		if timeLeft > 0 {
			cc.notify.Wait(timeLeft)
		}
		return false
	}, cc.delay, cc.duration)

	if !cc.interactions.AllHaveRequests() {
		for _, i := range cc.interactions.All() {
			if !i.HasRequests(1) {
				log.Infof("'%s' has no requests", i.Description)
			}
		}

		return c.JSON(http.StatusRequestTimeout, httpresponse.Error("timeout waiting for interactions to be met"))
	}
	return c.NoContent(http.StatusOK)
}

func indexHandler(c echo.Context) error {
	cc := c.(*ProxyContext)
	req := cc.Request()
	log.Infof("proxying %s %s", req.Method, req.URL.Path)

	mediaType, err := parseMediaTypeHeader(cc.Request().Header)
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("failed to parse Content-Type header. %s", err.Error()))
	}

	parseRequest, ok := supportedMediaTypes[mediaType]
	if !ok {
		return c.JSON(http.StatusUnsupportedMediaType, httpresponse.Errorf("unsupported Media Type: %s", mediaType))
	}

	allInteractions, ok := cc.interactions.FindAll(req.URL.Path, req.Method)
	if !ok {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to find interaction to Match '%s %s'", req.Method, req.URL.Path))
	}

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, httpresponse.Errorf("unable to read requestDocument data. %s", err.Error()))
	}

	err = cc.Request().Body.Close()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, httpresponse.Error(err.Error()))
	}

	cc.Request().Body = ioutil.NopCloser(bytes.NewBuffer(data))

	request, err := parseRequest(data, cc.Request().URL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, httpresponse.Errorf("unable to read requestDocument data. %s", err.Error()))
	}
	h := make(map[string]interface{})
	for headerName, headerValues := range cc.Request().Header {
		for _, headerValue := range headerValues {
			h[headerName] = headerValue
		}
	}
	request["headers"] = h

	unmatched := make(map[string][]string)
	matched := make([]*interaction, 0)
	for _, interaction := range allInteractions {
		ok, info := interaction.EvaluateConstrains(request, cc.interactions)
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

	cc.notify.Notify()
	cc.proxy.ServeHTTP(&ResponseModificationWriter{res: cc.Response(), interactions: matched}, req)
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
