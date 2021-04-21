package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/form3tech/pact-proxy/pkg/pactproxy"
	"github.com/pact-foundation/pact-go/dsl"
)

type ProxyStage struct {
	t               *testing.T
	pact            *dsl.Pact
	proxy           *pactproxy.PactProxy
	constraintValue string
	pactResult      error
	requestsToSend  int32
	requestsSent    int32
}

const (
	PostNamePact    = "A request to create a user"
	PostAddressPact = "A request to create an address"
)

func NewProxyStage(t *testing.T) (*ProxyStage, *ProxyStage, *ProxyStage, func()) {
	pact := &dsl.Pact{
		Consumer: "MyConsumer",
		Provider: "MyProvider",
		Host:     "localhost",
	}

	pact.Setup(true)
	proxy, err := pactproxy.
		Configuration(adminURL.String()).
		SetupProxy(proxyURL.String(), fmt.Sprintf("http://%s:%d", pact.Host, pact.Server.Port))

	if err != nil {
		t.Logf("Error setting up proxy: %v", err)
		t.Fail()
	}

	pact.Server.Port, err = strconv.Atoi(proxyURL.Port())
	if err != nil {
		t.Logf("Error parsing server port: %v", err)
		t.Fail()
	}

	stage := &ProxyStage{
		t:     t,
		proxy: proxy,
		pact:  pact,
	}

	return stage, stage, stage, func() {
		pactproxy.Configuration(adminURL.String()).Reset()
		pact.Teardown()
	}
}

func (s *ProxyStage) and() *ProxyStage {
	return s
}

func (s *ProxyStage) a_pact_that_allows_any_names() *ProxyStage {
	s.pact.
		AddInteraction().
		UponReceiving(PostNamePact).
		WithRequest(dsl.Request{
			Method:  "POST",
			Path:    dsl.String("/users"),
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
			Body:    dsl.MapMatcher{"name": dsl.Regex("any", ".*")},
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
			Body:    map[string]string{"name": "any"},
		})
	return s
}

func (s *ProxyStage) a_pact_that_allows_any_address() *ProxyStage {
	s.pact.
		AddInteraction().
		UponReceiving(PostAddressPact).
		WithRequest(dsl.Request{
			Method:  "POST",
			Path:    dsl.String("/addresses"),
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
			Body:    dsl.MapMatcher{"address": dsl.Regex("any", ".*")},
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
			Body:    map[string]string{"address": "any"},
		})
	return s
}

func (s *ProxyStage) a_constaint_is_added(name string) *ProxyStage {
	s.constraintValue = name
	return s
}

func (s *ProxyStage) a_request_is_sent_using_the_name(name string) {
	s.pactResult = s.pact.Verify(func() (err error) {
		s.proxy.ForInteraction(PostNamePact).AddConstraint("$.body.name", s.constraintValue)

		u := fmt.Sprintf("http://localhost:%s/users", proxyURL.Port())
		req, err := http.NewRequest("POST", u, strings.NewReader(fmt.Sprintf(`{"name":"%s"}`, name)))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")

		if _, err = http.DefaultClient.Do(req); err != nil {
			return err
		}

		return nil
	})
}

func (s *ProxyStage) pact_verification_is_successful() {
	if s.pactResult != nil {
		s.t.Error(s.pactResult)
		s.t.Fail()
	}
}

func (s *ProxyStage) pact_verification_is_not_successful() {
	if s.pactResult == nil {
		s.t.Error("pact verification did not fail")
		s.t.Fail()
	}
}

func (s *ProxyStage) multiple_requests_are_sent(requestsToSend int32) {
	s.pactResult = s.pact.Verify(func() (err error) {
		s.requestsToSend = requestsToSend
		atomic.StoreInt32(&s.requestsSent, 0)
		go func() {
			for i := int32(0); i < requestsToSend; i++ {
				u := fmt.Sprintf("http://localhost:%s/users", proxyURL.Port())
				req, err := http.NewRequest("POST", u, strings.NewReader(`{"name":"test"}`))
				if err != nil {
					s.t.Error(err)
					s.t.Fail()
				}

				req.Header.Set("Content-Type", "application/json")
				atomic.AddInt32(&s.requestsSent, 1)
				if _, err = http.DefaultClient.Do(req); err != nil {
					s.t.Error(err)
					s.t.Fail()
				}
			}
		}()

		if err := s.proxy.WaitForInteraction(PostNamePact, int(requestsToSend)); err != nil {
			s.t.Error(err)
			s.t.Fail()
		}

		return nil
	})
}

func (s *ProxyStage) the_proxy_waits_for_all_requests() *ProxyStage {
	sent := atomic.LoadInt32(&s.requestsSent)
	if sent != s.requestsToSend {
		s.t.Errorf("proxy did not wait for requests, sent=%d expected=%d", sent, s.requestsToSend)
		s.t.Fail()
	}
	return s
}

func (s *ProxyStage) requests_for_names_and_addresse_are_sent() *ProxyStage {
	s.pactResult = s.pact.Verify(func() (err error) {
		s.requestsToSend = 2
		atomic.StoreInt32(&s.requestsSent, 0)
		go func() {
			u := fmt.Sprintf("http://localhost:%s/users", proxyURL.Port())
			req, err := http.NewRequest("POST", u, strings.NewReader(`{"name":"test"}`))
			if err != nil {
				s.t.Error(err)
				s.t.Fail()
			}

			req.Header.Set("Content-Type", "application/json")
			atomic.AddInt32(&s.requestsSent, 1)
			if _, err = http.DefaultClient.Do(req); err != nil {
				s.t.Error(err)
				s.t.Fail()
			}

		}()

		go func() {
			u := fmt.Sprintf("http://localhost:%s/addresses", proxyURL.Port())
			req, err := http.NewRequest("POST", u, strings.NewReader(`{"address":"test"}`))
			if err != nil {
				s.t.Error(err)
				s.t.Fail()
			}

			req.Header.Set("Content-Type", "application/json")
			atomic.AddInt32(&s.requestsSent, 1)
			if _, err = http.DefaultClient.Do(req); err != nil {
				s.t.Error(err)
				s.t.Fail()
			}

		}()

		if err := s.proxy.WaitForAll(); err != nil {
			s.t.Error(err)
			s.t.Fail()
		}

		return nil
	})
	return s
}
