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
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/form3tech-oss/pact-proxy/pkg/pactproxy"
	"github.com/pact-foundation/pact-go/dsl"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type ProxyStage struct {
	t                     *testing.T
	assert                *assert.Assertions
	pact                  *dsl.Pact
	proxy                 *pactproxy.PactProxy
	contentTypeConstraint string
	nameConstraintValue   string
	bodyConstraintValue   string
	pactResult            error
	pactName              string
	requestsToSend        int32
	requestsSent          int32
	responses             []*http.Response
	responseBodies        [][]byte
	modifiedStatusCode    int
	modifiedAttempt       *int
	modifiedBody          map[string]interface{}
}

var largeString = strings.Repeat("long_string123BBmmF8BYezrBhCROOCRJfeH5k69hMKXH77TSvwF5GHUZFnbh1dsZ3d90HeR0jUIOovJJVS508uI17djeLFFSb7", 440)

func NewProxyStage(t *testing.T) (*ProxyStage, *ProxyStage, *ProxyStage) {
	proxy, err := setupAndWaitForProxy()
	if err != nil {
		t.Logf("Error setting up proxy: %v", err)
		t.Fail()
	}

	s := &ProxyStage{
		t:            t,
		assert:       assert.New(t),
		proxy:        proxy,
		pact:         pact,
		modifiedBody: make(map[string]interface{}),
		pactName:     "pact-" + strconv.FormatInt(time.Now().UnixMilli(), 10),
	}

	s.t.Cleanup(func() {
		pactproxy.Configuration(adminURL.String()).Reset()
	})

	return s, s, s
}

func setupAndWaitForProxy() (*pactproxy.PactProxy, error) {
	proxy, err := pactproxy.
		Configuration(adminURL.String()).
		SetupProxy(proxyURL.String(), fmt.Sprintf("http://%s:%d", pact.Host, originalPactServerPort))
	if err != nil {
		return nil, errors.Wrap(err, "proxy setup failed")
	}

	retryOpts := []retry.Option{
		retry.Attempts(10),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(500 * time.Millisecond),
	}

	err = retry.Do(proxy.IsReady, retryOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "proxy readiness wait failed")
	}

	return proxy, nil
}

func (s *ProxyStage) and() *ProxyStage {
	return s
}

func (s *ProxyStage) a_pact_that_allows_any_names() *ProxyStage {
	s.pact.
		AddInteraction().
		UponReceiving(s.pactName).
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

func (s *ProxyStage) a_pact_that_allows_any_age() *ProxyStage {
	s.pact.
		AddInteraction().
		UponReceiving(s.pactName).
		WithRequest(dsl.Request{
			Method:  "POST",
			Path:    dsl.String("/users"),
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
			Body:    dsl.MapMatcher{"age": dsl.Integer()},
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
			Body:    map[string]int64{"age": 100},
		})
	return s
}

func (s *ProxyStage) a_pact_that_allows_any_names_and_returns_no_body() *ProxyStage {
	s.pact.
		AddInteraction().
		UponReceiving(s.pactName).
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

func (s *ProxyStage) a_pact_that_allows_any_names_and_returns_large_body_and_any_name() *ProxyStage {
	s.pact.
		AddInteraction().
		UponReceiving(s.pactName).
		WithRequest(dsl.Request{
			Method:  "POST",
			Path:    dsl.String("/users"),
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
			Body:    dsl.MapMatcher{"name": dsl.Regex("any", ".*")},
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
			Body: dsl.MapMatcher{
				"large_string": dsl.String(largeString),
				"name":         dsl.String("any"),
			},
		})
	return s
}

func (s *ProxyStage) a_pact_that_allows_any_first_and_last_names() *ProxyStage {
	s.pact.
		AddInteraction().
		UponReceiving(s.pactName).
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

func (s *ProxyStage) a_pact_that_expects_plain_text() *ProxyStage {
	s.pact.
		AddInteraction().
		UponReceiving(s.pactName).
		WithRequest(dsl.Request{
			Method:  "POST",
			Path:    dsl.String("/users"),
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("text/plain")},
			Body:    "text",
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("text/plain")},
			Body:    "text",
		})
	return s
}

func (s *ProxyStage) a_pact_that_expects_plain_text_with_request_response(req, resp string) *ProxyStage {
	s.pact.
		AddInteraction().
		UponReceiving(s.pactName).
		WithRequest(dsl.Request{
			Method:  "POST",
			Path:    dsl.String("/users"),
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("text/plain")},
			Body:    req,
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("text/plain")},
			Body:    resp,
		})
	return s
}

func (s *ProxyStage) a_pact_that_expects_plain_text_without_request_content_type_header() *ProxyStage {
	s.pact.
		AddInteraction().
		UponReceiving(s.pactName).
		WithRequest(dsl.Request{
			Method: "POST",
			Path:   dsl.String("/users"),
			Body:   "text",
		}).
		WillRespondWith(dsl.Response{
			Status:  200,
			Headers: dsl.MapMatcher{"Content-Type": dsl.String("text/plain")},
			Body:    "text",
		})
	return s
}

func (s *ProxyStage) a_name_constraint_is_added(name string) *ProxyStage {
	s.nameConstraintValue = name
	return s
}

func (s *ProxyStage) a_content_type_constraint_is_added(value string) *ProxyStage {
	s.contentTypeConstraint = value
	return s
}

func (s *ProxyStage) a_body_constraint_is_added(name string) *ProxyStage {
	s.bodyConstraintValue = name
	return s
}

func (s *ProxyStage) a_modified_response_status_of_(statusCode int) *ProxyStage {
	s.modifiedStatusCode = statusCode
	return s
}

func (s *ProxyStage) a_modified_response_body_of_(path string, value interface{}) *ProxyStage {
	s.modifiedBody[path] = value
	return s
}

func (s *ProxyStage) a_modified_response_attempt_of(i int) {
	s.modifiedAttempt = &i
}

func (s *ProxyStage) a_plain_text_request_is_sent() {
	s.a_plain_text_request_is_sent_with_body("text")
}

func (s *ProxyStage) a_plain_text_request_is_sent_with_body(body string) {
	s.n_requests_are_sent_using_the_body_and_content_type(1, body, "text/plain")
}

func (s *ProxyStage) a_request_is_sent_using_the_name(name string) {
	s.n_requests_are_sent_using_the_name(1, name)
}

func (s *ProxyStage) n_requests_are_sent_using_the_name(n int, name string) {
	s.n_requests_are_sent_using_the_body(n, fmt.Sprintf(`{"name":"%s"}`, name))
}

func (s *ProxyStage) n_requests_are_sent_using_the_age(n int, age int64) {
	s.n_requests_are_sent_using_the_body(n, fmt.Sprintf(`{"age": %d}`, age))
}

func (s *ProxyStage) n_requests_are_sent_using_the_body(n int, body string) {
	s.n_requests_are_sent_using_the_body_and_content_type(n, body, "application/json")
}

func (s *ProxyStage) n_requests_are_sent_using_the_body_and_content_type(n int, body, contentType string) {
	s.pactResult = s.pact.Verify(func() error {
		i := s.proxy.
			ForInteraction(s.pactName)

		if s.nameConstraintValue != "" {
			i.AddConstraint("$.body.name", s.nameConstraintValue)
		}

		if s.contentTypeConstraint != "" {
			i.AddConstraint("$.headers[\"Content-Type\"]", s.contentTypeConstraint)
		}

		if s.bodyConstraintValue != "" {
			i.AddConstraint("$.body", s.bodyConstraintValue)
		}

		if s.modifiedStatusCode != 0 {
			i.AddModifier("$.status", fmt.Sprintf("%d", s.modifiedStatusCode), s.modifiedAttempt)
		}

		if len(s.modifiedBody) > 0 {
			for path, value := range s.modifiedBody {
				i.AddModifier(path, value, s.modifiedAttempt)
			}
		}

		u := fmt.Sprintf("http://localhost:%s/users", proxyURL.Port())
		for i := 0; i < n; i++ {
			if err := s.send_post_request_and_collect_response(body, u, contentType); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ProxyStage) send_post_request_and_collect_response(body, url, contentType string) error {
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	s.assert.NoError(err, "request creation failed")

	req.Header.Set("Content-Type", contentType)
	res, err := http.DefaultClient.Do(req)
	s.assert.NoError(err, "sending request failed")

	s.responses = append(s.responses, res)
	bodyBytes, err := io.ReadAll(res.Body)
	s.assert.NoError(err, "unable to read response body")
	s.responseBodies = append(s.responseBodies, bodyBytes)
	return nil
}

func (s *ProxyStage) multiple_requests_are_sent(requestsToSend int32) {
	s.pactResult = s.pact.Verify(func() (err error) {
		s.requestsToSend = requestsToSend
		atomic.StoreInt32(&s.requestsSent, 0)
		go func() {
			for i := int32(0); i < requestsToSend; i++ {
				u := fmt.Sprintf("http://localhost:%s/users", proxyURL.Port())
				req, err := http.NewRequest("POST", u, strings.NewReader(`{"name":"test"}`))
				s.assert.NoError(err)

				req.Header.Set("Content-Type", "application/json")
				atomic.AddInt32(&s.requestsSent, 1)
				_, err = http.DefaultClient.Do(req)
				s.assert.NoError(err)
			}
		}()

		err = s.proxy.WaitForInteraction(s.pactName, int(requestsToSend))
		s.assert.NoError(err)
		return nil
	})
}

func (s *ProxyStage) pact_verification_is_successful() *ProxyStage {
	s.assert.Nil(s.pactResult)
	return s
}

func (s *ProxyStage) pact_verification_is_not_successful() *ProxyStage {
	s.assert.NotNil(s.pactResult, "pact verification did not fail")
	return s
}

func (s *ProxyStage) the_proxy_waits_for_all_requests() *ProxyStage {
	sent := atomic.LoadInt32(&s.requestsSent)
	s.assert.Equal(s.requestsToSend, sent, "proxy did not wait for requests")
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

func (s *ProxyStage) the_nth_response_is_(n, statusCode int) *ProxyStage {
	s.assert.GreaterOrEqual(len(s.responses), n, "number of responses is less than expected")
	s.assert.Equalf(statusCode, s.responses[n-1].StatusCode, "Expected status code on attempt %d: %d, got : %d", n, statusCode, s.responses[n-1].StatusCode)
	return s
}

func (s *ProxyStage) the_nth_response_name_is_(n int, name string) *ProxyStage {
	s.assert.GreaterOrEqual(len(s.responses), n, "number of responses is less than expected")

	var body map[string]string
	err := json.Unmarshal(s.responseBodies[n-1], &body)
	s.assert.NoError(err, "unable to parse response body, %v", err)
	s.assert.Equalf(name, body["name"], "Expected name on attempt %d,: %s, got: %s", n, name, body["name"])
	return s
}

func (s *ProxyStage) the_nth_response_age_is_(n int, age int64) *ProxyStage {
	s.assert.GreaterOrEqual(len(s.responses), n, "number of responses is less than expected")

	var body map[string]int64
	err := json.Unmarshal(s.responseBodies[n-1], &body)
	s.assert.NoError(err, "unable to parse response body")

	s.assert.Equalf(age, body["age"], "Expected name on attempt %d,: %d, got: %d", n, age, body["age"])

	return s
}

func (s *ProxyStage) the_nth_response_body_has_(n int, key, value string) *ProxyStage {
	s.assert.GreaterOrEqual(len(s.responseBodies), n, "number of request bodies is les than expected")

	var responseBody map[string]string
	err := json.Unmarshal(s.responseBodies[n-1], &responseBody)
	s.assert.NoError(err, "unable to parse response body, %v", err)
	s.assert.Equalf(value, responseBody[key], "Expected %s on attempt %d,: %s, got: %s", key, n, value, responseBody[key])
	return s
}

func (s *ProxyStage) the_response_body_is(data []byte) *ProxyStage {
	return s.the_nth_response_body_is(1, data)
}

func (s *ProxyStage) the_response_body_to_plain_text_request_is_correct() *ProxyStage {
	return s.the_nth_response_body_is(1, []byte("text"))
}

func (s *ProxyStage) the_nth_response_body_is(n int, data []byte) *ProxyStage {
	s.assert.GreaterOrEqual(len(s.responseBodies), n, "number of request bodies is les than expected")

	body := s.responseBodies[n-1]
	c := bytes.Compare(body, data)
	s.assert.Equal(0, c, "Expected body did not match")
	return s
}

func (s *ProxyStage) n_responses_were_received(n int) *ProxyStage {
	s.assert.Len(s.responses, n)
	return s
}

func (s *ProxyStage) pact_can_be_generated() {
	u := fmt.Sprintf("http://localhost:%s/pact", proxyURL.Port())
	req, err := http.NewRequestWithContext(context.Background(), "POST", u, bytes.NewReader([]byte("{\"pact_specification_version\":\"3.0.0\"}")))
	s.assert.NoError(err)

	req.Header.Add("X-Pact-Mock-Service", "true")
	req.Header.Add("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	s.assert.NoError(err)
	defer resp.Body.Close()
	s.assert.Equal(http.StatusOK, resp.StatusCode, "Expected 200 but returned %d status code", resp.StatusCode)
}
