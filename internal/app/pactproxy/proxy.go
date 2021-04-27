package pactproxy

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/form3tech-oss/pact-proxy/internal/app/httpresponse"
	log "github.com/sirupsen/logrus"
)

func StartProxy(server *http.ServeMux, target *url.URL) {
	interactions := &Interactions{}

	proxy := httputil.NewSingleHostReverseProxy(target)
	notify := NewNotify()

	server.HandleFunc("/interactions/verification", func(res http.ResponseWriter, req *http.Request) {
		proxy.ServeHTTP(res, req)
	})

	server.HandleFunc("/interactions/constraints", func(res http.ResponseWriter, req *http.Request) {
		constraintBytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			httpresponse.Errorf(res, http.StatusBadRequest, "unable to read constraint. %s", err.Error())
			return
		}

		constraint, err := LoadConstraint(constraintBytes)
		if err != nil {
			httpresponse.Errorf(res, http.StatusBadRequest, "unable to read constraint. %s", err.Error())
			return
		}

		interaction, ok := interactions.Load(constraint.Interaction)
		if !ok {
			httpresponse.Errorf(res, http.StatusBadRequest, "unable to find interaction. %s", constraint.Interaction)
			return
		}

		log.Infof("adding constraint to interaction '%s'", interaction.Description)
		interaction.AddConstraint(constraint)
	})

	server.HandleFunc("/session", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			log.Infof("deleting session for %s", target)
			proxy.ServeHTTP(res, req)
			return
		}
	})

	server.HandleFunc("/interactions", func(res http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			proxy.ServeHTTP(res, req)
			interactions.Clear()
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
				httpresponse.Errorf(res, http.StatusBadRequest, "unable to read interaction. %s", err.Error())
				return
			}

			if interaction.Alias != "" {
				log.Infof("storing interaction '%s' (%s)", interaction.Description, interaction.Alias)
			} else {
				log.Infof("storing interaction '%s'", interaction.Description)
			}

			interactions.Store(interaction)

			err = req.Body.Close()
			if err != nil {
				httpresponse.Errorf(res, http.StatusInternalServerError, err.Error())
				return
			}

			req.Body = ioutil.NopCloser(bytes.NewBuffer(data))

			proxy.ServeHTTP(res, req)
			return
		}
	})

	server.HandleFunc("/interactions/wait", func(res http.ResponseWriter, req *http.Request) {
		waitFor := req.URL.Query().Get("interaction")
		waitForCount, err := strconv.Atoi(req.URL.Query().Get("count"))
		if err != nil {
			waitForCount = 1
		}

		if waitFor != "" {
			interaction, ok := interactions.Load(waitFor)
			if !ok {
				httpresponse.Errorf(res, http.StatusBadRequest, "cannot wait for interaction '%s', interaction not found.", waitFor)
			}

			log.Infof("waiting for %s", waitFor)
			retryFor(func(timeLeft time.Duration) bool {
				if interaction.HasRequests(waitForCount) {
					return true
				}
				if timeLeft > 0 {
					notify.Wait(timeLeft)
				}
				return false
			}, 500*time.Millisecond, 15*time.Second)

			if !interaction.HasRequests(waitForCount) {
				httpresponse.Error(res, http.StatusRequestTimeout, "timeout waiting for interactions to be met")
			}
			return
		}

		log.Info("waiting for all")
		retryFor(func(timeLeft time.Duration) bool {
			if interactions.AllHaveRequests() {
				return true
			}
			if timeLeft > 0 {
				notify.Wait(timeLeft)
			}
			return false
		}, 500*time.Millisecond, 15*time.Second)

		if !interactions.AllHaveRequests() {
			for _, i := range interactions.All() {
				if !i.HasRequests(1) {
					log.Infof("'%s' has no requests", i.Description)
				}
			}

			httpresponse.Error(res, http.StatusRequestTimeout, "timeout waiting for interactions to be met")
		}
	})

	server.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		log.Infof("proxying %s", req.URL.Path)
		allInteractions, ok := interactions.FindAll(req.URL.Path, req.Method)
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

		request, err := LoadRequest(data, req.URL)
		if err != nil {
			httpresponse.Errorf(res, http.StatusInternalServerError, "unable to read requestDocument data. %s", err.Error())
			return
		}

		unmatched := make(map[string][]string)
		for _, interaction := range allInteractions {
			ok, info := interaction.EvaluateConstrains(request, interactions)
			if ok {
				log.Infof("interaction '%s' matched to '%s %s'", interaction.Description, req.Method, req.URL.Path)
				interaction.StoreRequest(request)
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

		notify.Notify()
		proxy.ServeHTTP(res, req)
	})
}
