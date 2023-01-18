package configuration

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"testing"

	"github.com/form3tech-oss/pact-proxy/internal/app/pactproxy"
	"github.com/stretchr/testify/require"
)

func TestConfigureProxy_Port(t *testing.T) {
	defer CloseAllServers()

	serverAddrs := []*url.URL{}
	names := []string{"foo", "bar"}
	for _, name := range names {
		safeName := name

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello, "+safeName)
		}))
		defer ts.Close()

		url1, err := url.Parse(ts.URL)
		require.NoError(t, err)

		serverAddr, err := getFreePortURL()
		require.NoError(t, err)

		err = ConfigureProxy(pactproxy.Config{ServerAddress: *serverAddr, Target: *url1})
		require.NoError(t, err)

		serverAddrs = append(serverAddrs, serverAddr)
	}

	for i, addr := range serverAddrs {
		res, err := http.Get(addr.String() + "/pact")
		require.NoError(t, err)

		greeting, err := io.ReadAll(res.Body)
		res.Body.Close()
		require.NoError(t, err)

		expected := fmt.Sprintf("Hello, %s\n", names[i])
		require.Equal(t, expected, string(greeting))
	}
}

func TestProxyConfig_Path(t *testing.T) {
	defer CloseAllServers()

	serverAddrs := []*url.URL{}
	names := []string{"foo", "bar"}
	for _, name := range names {
		safeName := name

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello, "+safeName)
		}))
		defer ts.Close()

		url1, err := url.Parse(ts.URL + "/" + safeName)
		require.NoError(t, err)

		serverAddr, err := getFreePortURL()
		require.NoError(t, err)

		serverAddr.Path = "/" + safeName
		err = ConfigureProxy(pactproxy.Config{ServerAddress: *serverAddr, Target: *url1})
		require.NoError(t, err)

		serverAddrs = append(serverAddrs, serverAddr)
	}

	for i, addr := range serverAddrs {

		addr.Path = path.Join(addr.Path, "/pact")
		res, err := http.Get(addr.String())
		require.NoError(t, err)

		greeting, err := io.ReadAll(res.Body)
		res.Body.Close()
		require.NoError(t, err)

		expected := fmt.Sprintf("Hello, %s\n", names[i])
		require.Equal(t, expected, string(greeting))
	}
}

func getFreePortURL() (*url.URL, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	defer l.Close()
	urlStr := fmt.Sprintf("http://localhost:%d", l.Addr().(*net.TCPAddr).Port)
	return url.Parse(urlStr)
}
