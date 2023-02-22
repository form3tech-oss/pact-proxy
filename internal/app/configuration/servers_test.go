package configuration

import (
	"context"
	"net/url"
	"testing"

	"github.com/form3tech-oss/pact-proxy/internal/app/pactproxy"
	"github.com/stretchr/testify/require"
)

// This test ensures that the server will start up correctly (or error)
// for different combinations of host, port and path.
func TestGetServer(t *testing.T) {
	type testCase struct {
		name        string
		url1        string
		url2        string
		shouldError bool
	}

	for _, tc := range []testCase{
		{
			name:        "Same host, same port, same path",
			url1:        "http://:8080/foo",
			url2:        "http://:8080/foo",
			shouldError: true,
		},
		{
			name:        "Same host, same port, different path",
			url1:        "http://:8080/foo",
			url2:        "http://:8080/bar",
			shouldError: false,
		},
		{
			name:        "Same host, same port, nested path",
			url1:        "http://:8080/foo",
			url2:        "http://:8080/foo/bar",
			shouldError: false,
		},
		{
			name:        "Same host, different port, same path",
			url1:        "http://:8080/foo",
			url2:        "http://:8081/foo",
			shouldError: false,
		},
		{
			name:        "Same host, different port, different path",
			url1:        "http://:8080/foo",
			url2:        "http://:8081/bar",
			shouldError: false,
		},
		{
			name:        "Different host, same port, same path",
			url1:        "http://localhost:8080/foo",
			url2:        "http://test:8080/foo",
			shouldError: false,
		},
		{
			name:        "Different host, same port, different path",
			url1:        "http://localhost:8080/foo",
			url2:        "http://test:8080/bar",
			shouldError: false,
		},
		{
			name:        "Different host, different port, same path",
			url1:        "http://localhost:8080/foo",
			url2:        "http://test:8081/foo",
			shouldError: false,
		},
		{
			name:        "Different host, different port, different path",
			url1:        "http://localhost:8080/foo",
			url2:        "http://test:8081/bar",
			shouldError: false,
		},
		{
			name:        "Same host, same port, no path",
			url1:        "http://:8080/",
			url2:        "http://:8080/",
			shouldError: true,
		},
		{
			name:        "Same host, different port, no path",
			url1:        "http://:8080/",
			url2:        "http://:8081/",
			shouldError: false,
		},
	} {
		t.Run(tc.name, func(st *testing.T) {
			defer ShutdownAllServers(context.Background())

			url1, err := url.Parse(tc.url1)
			require.NoError(t, err)

			url2, err := url.Parse(tc.url2)
			require.NoError(t, err)

			err = StartServer(url1, &pactproxy.Config{})
			require.NoError(t, err)

			err = StartServer(url2, &pactproxy.Config{})
			require.Equalf(t, tc.shouldError, err != nil, "found error: %s", err)
		})
	}
}
