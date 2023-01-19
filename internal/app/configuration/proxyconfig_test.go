package configuration

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/form3tech-oss/pact-proxy/internal/app/pactproxy"
	"github.com/pact-foundation/pact-go/utils"
	"github.com/stretchr/testify/require"
)

// This test ensures that the correct proxy backend is called, and correct response returned
// for two proxy backends listening on different ports.
func TestConfigureProxy_Port(t *testing.T) {
	defer ShutdownAllServers(context.Background())

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

// This test ensures that the correct proxy backend is called, and correct response returned
// for two proxy backends listening on different paths.
func TestProxyConfig_Path(t *testing.T) {
	defer ShutdownAllServers(context.Background())

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

func TestConfigureProxy_TLS(t *testing.T) {
	defer ShutdownAllServers(context.Background())

	defer cleanupCertificates()
	err := createCertificates()
	require.NoError(t, err)

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

		port, err := utils.GetFreePort()
		require.NoError(t, err)

		serverAddr := url.URL{
			Scheme: "https",
			Host:   fmt.Sprintf("localhost:%d", port),
		}

		config := pactproxy.Config{
			ServerAddress: serverAddr,
			Target:        *url1,
			TLSCAFile:     "test_ca.pem",
			TLSCertFile:   "test_client.pem",
			TLSKeyFile:    "test_client.key",
		}

		err = ConfigureProxy(config)
		require.NoError(t, err)

		serverAddrs = append(serverAddrs, &serverAddr)
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

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

// gets a free port on the localhost and returns it as a url.
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

func createCertificates() error {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Form3"},
			Country:       []string{"GB"},
			Province:      []string{""},
			Locality:      []string{"London"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization:  []string{"Form3"},
			Country:       []string{"GB"},
			Province:      []string{""},
			Locality:      []string{"London"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		DNSNames:            []string{"localhost"},
		IPAddresses:         []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:           time.Now(),
		NotAfter:            time.Now().AddDate(10, 0, 0),
		SubjectKeyId:        []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:         []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:            x509.KeyUsageDigitalSignature,
		PermittedDNSDomains: []string{"localhost"},
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})

	os.WriteFile("test_ca.pem", caPEM.Bytes(), 0644)
	os.WriteFile("test_client.pem", certPEM.Bytes(), 0644)
	os.WriteFile("test_client.key", certPrivKeyPEM.Bytes(), 0644)
	// os.WriteFile("test_client.key", x509.MarshalPKCS1PrivateKey(certPrivKey), 0644)

	return nil
}

func cleanupCertificates() {
	// os.Remove("test_ca.pem")
	// os.Remove("test_client.pem")
	// os.Remove("test_client.key")
}
