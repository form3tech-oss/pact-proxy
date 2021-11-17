package pactproxy_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/form3tech-oss/pact-proxy/pkg/pactproxy"
	"github.com/stretchr/testify/assert"
)

func TestConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		expectedBody string
		body         string
		err          error
	}{
		{
			name:         "basic creation",
			method:       "POST",
			expectedBody: "{}",
			body:         "",
			err:          nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/proxies", r.URL.Path)

				body, err := ioutil.ReadAll(r.Body)
				assert.NoError(t, err)
				_ = body

				rw.Write([]byte("10.0.0.1"))
			}))
			defer ts.Close()

			conf := pactproxy.Configuration(ts.URL)

			_, err := conf.SetupProxy("127.0.0.1:80", "127.0.0.2:80")
			assert.ErrorIs(t, err, tt.err)
		})
	}
}
