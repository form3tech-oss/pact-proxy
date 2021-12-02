package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/form3tech-oss/pact-proxy/pkg/pactproxy"
	"github.com/pact-foundation/pact-go/dsl"
)

type ProxyStage struct {
	t                  *testing.T
	pact               *dsl.Pact
	proxy              *pactproxy.PactProxy
	constraintValue    string
	pactResult         error
	requestsToSend     int32
	requestsSent       int32
	responses          []*http.Response
	responseBodies     [][]byte
	modifiedStatusCode int
	modifiedAttempt    *int
	modifiedBody       map[string]string
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
		t:            t,
		proxy:        proxy,
		pact:         pact,
		modifiedBody: make(map[string]string),
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

func (s *ProxyStage) a_pact_that_returns_no_body() *ProxyStage {
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
			Status:  204,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
		})
	return s
}

func (s *ProxyStage) a_pact_that_allows_any_first_and_last_names() *ProxyStage {
	s.pact.
		AddInteraction().
		UponReceiving(PostNamePact).
		WithRequest(dsl.Request{
			Method:  "POST",
			Path:    dsl.String("/users"),
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
			Body: dsl.MapMatcher{
				"first_name": dsl.Regex("any", ".*"),
				"last_name":  dsl.Regex("any", ".*"),
			},
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
			Body:    map[string]string{"first_name": "any", "last_name": "any"},
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

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		s.responses = append(s.responses, res)
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			s.t.Fatalf("unable to read response body, %v", err)
		}
		s.responseBodies = append(s.responseBodies, bodyBytes)
		return nil
	})
}

func (s *ProxyStage) a_request_is_sent_with_modifiers_using_the_name(name string) {
	s.n_requests_are_sent_with_modifiers_using_the_name(1, name)
}

func (s *ProxyStage) n_requests_are_sent_with_modifiers_using_the_name(n int, name string) {
	s.n_requests_are_sent_with_modifiers_using_the_body(n, fmt.Sprintf(`{"name":"%s"}`, name))
}

func (s *ProxyStage) n_requests_are_sent_with_modifiers_using_the_body(n int, body string) {
	s.pactResult = s.pact.Verify(func() (err error) {
		i := s.proxy.
			ForInteraction(PostNamePact)

		if s.modifiedStatusCode != 0 {
			i.AddModifier("$.status", fmt.Sprintf("%d", s.modifiedStatusCode), s.modifiedAttempt)
		}

		for path, value := range s.modifiedBody {
			i.AddModifier(path, value, s.modifiedAttempt)
		}

		u := fmt.Sprintf("http://localhost:%s/users", proxyURL.Port())

		for i := 0; i < n; i++ {
			req, err := http.NewRequest("POST", u, strings.NewReader(body))
			if err != nil {
				return err
			}

			req.Header.Set("Content-Type", "application/json")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				s.t.Error(err)
				return err
			}
			s.responses = append(s.responses, res)
			bodyBytes, err := io.ReadAll(res.Body)
			if err != nil {
				s.t.Fatalf("unable to read response body, %v", err)
			}
			s.responseBodies = append(s.responseBodies, bodyBytes)
		}

		return nil
	})
}

func (s *ProxyStage) pact_verification_is_successful() *ProxyStage {
	if s.pactResult != nil {
		s.t.Error(s.pactResult)
		s.t.Fail()
	}

	return s
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

func (s *ProxyStage) a_modified_response_status_of_(statusCode int) *ProxyStage {
	s.modifiedStatusCode = statusCode
	return s
}

func (s *ProxyStage) a_modified_response_body_of_(path, value string) *ProxyStage {
	s.modifiedBody[path] = value
	return s
}

func (s *ProxyStage) the_response_is_(statusCode int) *ProxyStage {
	s.the_nth_response_is_(1, statusCode)

	return s
}

func (s *ProxyStage) the_response_name_is_(name string) *ProxyStage {
	s.the_nth_response_name_is_(1, name)

	return s
}

func (s *ProxyStage) a_modified_response_attempt_of(i int) {
	s.modifiedAttempt = &i
}

func (s *ProxyStage) the_nth_response_is_(n, statusCode int) *ProxyStage {
	if len(s.responses) < n {
		s.t.Fatalf("Expected at least %d responses, got %d", n, len(s.responses))
	}

	if s.responses[n-1].StatusCode != statusCode {
		s.t.Fatalf("Expected status code on attemt %d: %d, got : %d", n, statusCode, s.responses[n-1].StatusCode)
	}

	return s
}

func (s *ProxyStage) the_nth_response_name_is_(n int, name string) *ProxyStage {
	if len(s.responses) < n {
		s.t.Fatalf("Expected at least %d responses, got %d", n, len(s.responses))
	}

	var body map[string]string
	if err := json.Unmarshal(s.responseBodies[n-1], &body); err != nil {
		s.t.Fatalf("unable to parse response body, %v", err)
	}

	if body["name"] != name {
		s.t.Fatalf("Expected name on attempt %d,: %s, got: %s", n, name, body["name"])
	}

	return s
}

func (s *ProxyStage) the_nth_response_body_has_(n int, key, value string) *ProxyStage {
	if len(s.responseBodies) < n {
		s.t.Fatalf("Expected at least %d responses, got %d", n, len(s.responseBodies))
	}

	var responseBody map[string]string
	if err := json.Unmarshal(s.responseBodies[n-1], &responseBody); err != nil {
		s.t.Fatalf("unable to parse response body, %v", err)
	}

	if responseBody[key] != value {
		s.t.Fatalf("Expected %s on attempt %d,: %s, got: %s", key, n, value, responseBody[key])
	}

	return s
}

func (s *ProxyStage) n_responses_were_received(n int) *ProxyStage {
	count := len(s.responses)
	if count != n {
		s.t.Fatalf("Expected %d responses, got %d", n, count)
	}

	return s
}

func (s *ProxyStage) pact_can_be_generated() {
	u := fmt.Sprintf("http://localhost:%s/pact", proxyURL.Port())
	req, err := http.NewRequestWithContext(context.Background(), "POST", u, bytes.NewReader([]byte("{\"pact_specification_version\":\"3.0.0\"}")))
	if err != nil {
		s.t.Error(err)
		return
	}

	req.Header.Add("X-Pact-Mock-Service", "true")
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.t.Error(err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		s.t.Fatalf("Expected 200 but returned %d status code", resp.StatusCode)
	}

	defer func() { _ = resp.Body.Close() }()
}
