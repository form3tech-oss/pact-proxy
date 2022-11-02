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
	Target        url.URL       // Do not load Target from env, we set this for each value from Proxies
}

var supportedMediaTypes = map[string]func([]byte, *url.URL) (requestDocument, error){
	mediaTypeJSON: ParseJSONRequest,
	mediaTypeText: ParsePlainTextRequest,
}

func StartProxy(server *http.ServeMux, config *Config) {
	api := api{
		target:       &config.Target,
		proxy:        httputil.NewSingleHostReverseProxy(&config.Target),
		interactions: &Interactions{},
		notify:       NewNotify(),
		delay:        config.WaitDelay,
		duration:     config.WaitDuration,
	}
	if api.delay == 0 {
		api.delay = defaultDelay
	}
	if api.duration == 0 {
		api.duration = defaultDuration
	}

	for path, handler := range map[string]func(http.ResponseWriter, *http.Request){
		"/ready":                     api.readinessHandler,
		"/interactions/verification": api.proxyPassHandler,
		"/pact":                      api.proxyPassHandler,
		"/interactions/constraints":  api.interactionsConstraintsHandler,
		"/interactions/modifiers":    api.interactionsModifiersHandler,
		"/session":                   api.sessionHandler,
		"/interactions":              api.interactionsHandler,
		"/interactions/wait":         api.interactionsWaitHandler,
		"/":                          api.indexHandler,
	} {
		server.HandleFunc(path, handler)
	}
}

type api struct {
	target       *url.URL
	proxy        *httputil.ReverseProxy
	interactions *Interactions
	notify       *notify
	delay        time.Duration
	duration     time.Duration
}

func (a *api) proxyPassHandler(res http.ResponseWriter, req *http.Request) {
	a.proxy.ServeHTTP(res, req)
}

func (a *api) readinessHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *api) interactionsConstraintsHandler(res http.ResponseWriter, req *http.Request) {
	constraintBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		httpresponse.Errorf(res, http.StatusBadRequest, "unable to read constraint. %s", err.Error())
		return
	}

	constraint, err := LoadConstraint(constraintBytes)
	if err != nil {
		httpresponse.Errorf(res, http.StatusBadRequest, "unable to load constraint. %s", err.Error())
		return
	}

	interaction, ok := a.interactions.Load(constraint.Interaction)
	if !ok {
		httpresponse.Errorf(res, http.StatusBadRequest, "unable to find interaction. %s", constraint.Interaction)
		return
	}

	log.Infof("adding constraint to interaction '%s'", interaction.Description)
	interaction.AddConstraint(constraint)
}

func (a *api) interactionsModifiersHandler(res http.ResponseWriter, req *http.Request) {
	modifierBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		httpresponse.Errorf(res, http.StatusBadRequest, "unable to read modifier. %s", err.Error())
		return
	}

	modifier, err := loadModifier(modifierBytes)
	if err != nil {
		httpresponse.Errorf(res, http.StatusBadRequest, "unable to load modifier. %s", err.Error())
		return
	}

	interaction, ok := a.interactions.Load(modifier.Interaction)
	if !ok {
		httpresponse.Errorf(res, http.StatusBadRequest, "unable to find interaction for modifier. %s", modifier.Interaction)
		return
	}

	log.Infof("adding modifier to interaction '%s'", interaction.Description)
	interaction.Modifiers.AddModifier(modifier)
}

func (a *api) sessionHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodDelete {
		log.Infof("deleting session for %s", a.target)
		a.proxy.ServeHTTP(res, req)
		return
	}
}

func (a *api) interactionsHandler(res http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodDelete {
		log.Info("deleting interactions")
		a.proxy.ServeHTTP(res, req)
		a.interactions.Clear()
		return
	}

	if req.Method == http.MethodPost {
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			httpresponse.Errorf(res, http.StatusBadRequest, "unable to read interaction. %s", err.Error())
			return
		}

		interaction, err := LoadInteraction(data, req.URL.Query().Get("alias"))
		if err != nil {
			httpresponse.Errorf(res, http.StatusBadRequest, "unable to load interaction. %s", err.Error())
			return
		}

		if interaction.Alias != "" {
			log.Infof("storing interaction '%s' (%s)", interaction.Description, interaction.Alias)
		} else {
			log.Infof("storing interaction '%s'", interaction.Description)
		}

		a.interactions.Store(interaction)

		err = req.Body.Close()
		if err != nil {
			httpresponse.Errorf(res, http.StatusInternalServerError, err.Error())
			return
		}

		req.Body = ioutil.NopCloser(bytes.NewBuffer(data))

		a.proxy.ServeHTTP(res, req)
		return
	}
}

func (a *api) interactionsWaitHandler(res http.ResponseWriter, req *http.Request) {
	waitForCount, err := strconv.Atoi(req.URL.Query().Get("count"))
	if err != nil {
		waitForCount = 1
	}

	if waitFor := req.URL.Query().Get("interaction"); waitFor != "" {
		interaction, ok := a.interactions.Load(waitFor)
		if !ok {
			httpresponse.Errorf(res, http.StatusBadRequest, "cannot wait for interaction '%s', interaction not found.", waitFor)
			return
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
			httpresponse.Error(res, http.StatusRequestTimeout, "timeout waiting for interactions to be met")
		}

		return
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

		httpresponse.Error(res, http.StatusRequestTimeout, "timeout waiting for interactions to be met")
	}
}

func (a *api) indexHandler(res http.ResponseWriter, req *http.Request) {
	log.Infof("proxying %s %s", req.Method, req.URL.Path)

	mediaType, err := parseMediaTypeHeader(req.Header)
	if err != nil {
		httpresponse.Errorf(res, http.StatusBadRequest, "failed to parse Content-Type header. %s", err.Error())
		return
	}

	parseRequest, ok := supportedMediaTypes[mediaType]
	if !ok {
		httpresponse.Errorf(res, http.StatusUnsupportedMediaType, "unsupported Media Type: %s", mediaType)
		return
	}

	allInteractions, ok := a.interactions.FindAll(req.URL.Path, req.Method)
	if !ok {
		httpresponse.Errorf(res, http.StatusBadRequest, "unable to find interaction to Match '%s %s'", req.Method, req.URL.Path)
		return
	}

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		httpresponse.Errorf(res, http.StatusBadRequest, "unable to read requestDocument data. %s", err.Error())
		return
	}

	err = req.Body.Close()
	if err != nil {
		httpresponse.Error(res, http.StatusInternalServerError, err.Error())
		return
	}

	req.Body = ioutil.NopCloser(bytes.NewBuffer(data))

	request, err := parseRequest(data, req.URL)
	if err != nil {
		httpresponse.Errorf(res, http.StatusInternalServerError, "unable to read requestDocument data. %s", err.Error())
		return
	}
	h := make(map[string]interface{})
	for headerName, headerValues := range req.Header {
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
		httpresponse.Error(res, http.StatusBadRequest, "constraints do not match")
		return
	}

	a.notify.Notify()
	a.proxy.ServeHTTP(&ResponseModificationWriter{res: res, interactions: matched}, req)
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
