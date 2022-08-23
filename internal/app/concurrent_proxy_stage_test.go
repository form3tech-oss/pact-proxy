package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/form3tech-oss/pact-proxy/pkg/pactproxy"
	"github.com/pact-foundation/pact-go/dsl"
	log "github.com/sirupsen/logrus"
)

type ConcurrentProxyStage struct {
	t                                  *testing.T
	proxy                              *pactproxy.PactProxy
	pact                               *dsl.Pact
	modifiedNameStatusCode             int
	modifiedAddressStatusCode          int
	concurrentUserRequestsPerSecond    int
	concurrentUserRequestsDuration     time.Duration
	concurrentAddressRequestsPerSecond int
	concurrentAddressRequestsDuration  time.Duration
	userResponses                      []*http.Response
	addressResponses                   []*http.Response
}

func NewConcurrentProxyStage(t *testing.T) (*ConcurrentProxyStage, *ConcurrentProxyStage, *ConcurrentProxyStage) {
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
	s := &ConcurrentProxyStage{
		t:     t,
		pact:  pact,
		proxy: proxy,
	}

	t.Cleanup(func() {
		pactproxy.Configuration(adminURL.String()).Reset()
		pact.Teardown()
	})

	return s, s, s
}

func (s *ConcurrentProxyStage) and() *ConcurrentProxyStage {
	return s
}

func (s *ConcurrentProxyStage) a_modified_name_status_code() *ConcurrentProxyStage {
	s.modifiedNameStatusCode = http.StatusBadGateway
	return s
}

func (s *ConcurrentProxyStage) a_modified_address_status_code() *ConcurrentProxyStage {
	s.modifiedAddressStatusCode = http.StatusConflict
	return s
}

func (s *ConcurrentProxyStage) a_pact_that_allows_any_names() *ConcurrentProxyStage {
	s.pact.
		AddInteraction().
		UponReceiving(postNamePact).
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

func (s *ConcurrentProxyStage) a_pact_that_allows_any_address() *ConcurrentProxyStage {
	s.pact.
		AddInteraction().
		UponReceiving(postAddressPact).
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

func (s *ConcurrentProxyStage) x_concurrent_user_requests_per_second_are_made_for_y_seconds(x int, y time.Duration) *ConcurrentProxyStage {
	s.concurrentUserRequestsPerSecond = x
	s.concurrentUserRequestsDuration = y

	return s
}

func (s *ConcurrentProxyStage) x_concurrent_address_requests_per_second_are_made_for_y_seconds(x int, y time.Duration) *ConcurrentProxyStage {
	s.concurrentAddressRequestsPerSecond = x
	s.concurrentAddressRequestsDuration = y

	return s
}

func (s *ConcurrentProxyStage) the_concurrent_requests_are_sent() {
	err := s.pact.Verify(func() (err error) {
		s.proxy.ForInteraction(postNamePact).AddModifier("$.status", fmt.Sprintf("%d", s.modifiedNameStatusCode), nil)
		s.proxy.ForInteraction(postAddressPact).AddModifier("$.status", fmt.Sprintf("%d", s.modifiedAddressStatusCode), nil)

		wg := sync.WaitGroup{}

		wg.Add(1)
		go func() {
			defer wg.Done()
			sendConcurrentRequests(s.concurrentUserRequestsPerSecond, s.concurrentUserRequestsDuration, s.makeUserRequest)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			sendConcurrentRequests(s.concurrentAddressRequestsPerSecond, s.concurrentAddressRequestsDuration, s.makeAddressRequest)
		}()

		wg.Wait()

		return nil
	})

	if err != nil {
		s.t.Error(err)
	}
}

func (s *ConcurrentProxyStage) makeUserRequest() {
	u := fmt.Sprintf("http://localhost:%s/users", proxyURL.Port())
	req, err := http.NewRequest("POST", u, strings.NewReader(`{"name":"jim"}`))
	if err != nil {
		s.t.Error(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		s.t.Error(err)
		return
	}
	s.userResponses = append(s.userResponses, res)
}

func (s *ConcurrentProxyStage) makeAddressRequest() {
	u := fmt.Sprintf("http://localhost:%s/addresses", proxyURL.Port())
	req, err := http.NewRequest("POST", u, strings.NewReader(`{"address":"test"}`))
	if err != nil {
		s.t.Error(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		s.t.Error(err)
		return
	}
	s.addressResponses = append(s.addressResponses, res)
}

func sendConcurrentRequests(requests int, d time.Duration, f func()) {
	log.Infof("sending %d concurrent requests per second for %d", requests, d)
	timer := time.NewTimer(d)
	stop := make(chan bool)

	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				for i := 0; i < requests; i++ {
					f()
				}
			}
		}
	}()

	<-timer.C
	stop <- true
}

func (s *ConcurrentProxyStage) all_the_user_responses_should_have_the_right_status_code() *ConcurrentProxyStage {
	expectedLen := s.concurrentUserRequestsPerSecond * int(s.concurrentUserRequestsDuration/time.Second)
	if len(s.userResponses) != expectedLen {
		s.t.Errorf("expected %d user responses, but got %d", expectedLen, len(s.userResponses))
	}
	for _, res := range s.userResponses {
		if s.modifiedNameStatusCode != res.StatusCode {
			s.t.Errorf("expected user status code of %d, but got %d", s.modifiedNameStatusCode, res.StatusCode)
		}
	}

	return s
}

func (s *ConcurrentProxyStage) all_the_address_responses_should_have_the_right_status_code() *ConcurrentProxyStage {
	expectedLen := s.concurrentAddressRequestsPerSecond * int(s.concurrentAddressRequestsDuration/time.Second)
	if len(s.addressResponses) != expectedLen {
		s.t.Errorf("expected %d address responses, but got %d", expectedLen, len(s.addressResponses))
	}
	for _, res := range s.addressResponses {
		if s.modifiedAddressStatusCode != res.StatusCode {
			s.t.Errorf("expected address status code of %d, but got %d", s.modifiedAddressStatusCode, res.StatusCode)
		}
	}

	return s
}
