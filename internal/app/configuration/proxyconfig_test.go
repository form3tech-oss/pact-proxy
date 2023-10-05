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

const (
	testCAFile   = "test_ca.pem"
	testCertFile = "test_client.pem"
	testKeyFile  = "test_client.key"
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

func TestConfigureProxy_MTLS(t *testing.T) {
	defer ShutdownAllServers(context.Background())

	caPEM, clientPEM, clientPKeyPEM, err := createCertificates()
	require.NoError(t, err)

	defer cleanupCertificates()

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
			TLSCAFile:     testCAFile,
			TLSCertFile:   testCertFile,
			TLSKeyFile:    testKeyFile,
		}

		err = ConfigureProxy(config)
		require.NoError(t, err)

		serverAddrs = append(serverAddrs, &serverAddr)
	}

	rootCACertPool := x509.NewCertPool()
	ok := rootCACertPool.AppendCertsFromPEM(caPEM)
	require.True(t, ok)

	clientCert, err := tls.X509KeyPair(clientPEM, clientPKeyPEM)
	require.NoError(t, err)

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		RootCAs:      rootCACertPool,
		Certificates: []tls.Certificate{clientCert},
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

func TestProxyConfig_RejectUnrecognizedInteractions(t *testing.T) {
	os.Clearenv()
	expectedDefaultValue := true

	config, err := NewFromEnv()
	require.NoError(t, err)
	require.Equal(t, expectedDefaultValue, config.RejectUnrecognizedInteractions) // If no env var is specified it should default to true

	os.Setenv("REJECT_UNRECOGNIZED_INTERACTIONS", "false")
	config, err = NewFromEnv()
	require.NoError(t, err)
	require.Equal(t, false, config.RejectUnrecognizedInteractions)

	os.Setenv("REJECT_UNRECOGNIZED_INTERACTIONS", "true")
	config, err = NewFromEnv()
	require.NoError(t, err)
	require.Equal(t, true, config.RejectUnrecognizedInteractions)

	os.Setenv("REJECT_UNRECOGNIZED_INTERACTIONS", "not a boolean")
	config, err = NewFromEnv()
	require.Error(t, err) // Parse error

	os.Setenv("REJECT_UNRECOGNIZED_INTERACTIONS", "0")
	config, err = NewFromEnv()
	require.NoError(t, err)
	require.Equal(t, false, config.RejectUnrecognizedInteractions)

	os.Setenv("REJECT_UNRECOGNIZED_INTERACTIONS", "1")
	config, err = NewFromEnv()
	require.NoError(t, err)
	require.Equal(t, true, config.RejectUnrecognizedInteractions)

	os.Setenv("REJECT_UNRECOGNIZED_INTERACTIONS", "2")
	config, err = NewFromEnv()
	require.Error(t, err) // Parse error

	os.Setenv("REJECT_UNRECOGNIZED_INTERACTIONS", "-1")
	config, err = NewFromEnv()
	require.Error(t, err) // Parse error
}

// gets a free port on the localhost and returns it as a url.
func getFreePortURL() (*url.URL, error) {
	port, err := utils.GetFreePort()
	if err != nil {
		return nil, err
	}

	urlStr := fmt.Sprintf("http://localhost:%d", port)
	return url.Parse(urlStr)
}

func createCertificates() ([]byte, []byte, []byte, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization: []string{"Form3"},
			Country:      []string{"GB"},
			Locality:     []string{"London"},
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
		return nil, nil, nil, err
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, nil, err
	}

	caPEM := new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return nil, nil, nil, err
	}

	caPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	if err != nil {
		return nil, nil, nil, err
	}

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization: []string{"Form3"},
			Country:      []string{"GB"},
			Locality:     []string{"London"},
		},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, nil, err
	}

	certPEM := new(bytes.Buffer)
	err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return nil, nil, nil, err
	}

	certPrivKeyPEM := new(bytes.Buffer)
	err = pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	if err != nil {
		return nil, nil, nil, err
	}

	certOutput := map[string][]byte{
		testCAFile:   caPEM.Bytes(),
		testCertFile: certPEM.Bytes(),
		testKeyFile:  certPrivKeyPEM.Bytes(),
	}

	for file, content := range certOutput {
		if err := os.WriteFile(file, content, 0644); err != nil {
			return nil, nil, nil, err
		}
	}

	return caPEM.Bytes(), certPEM.Bytes(), certPrivKeyPEM.Bytes(), nil
}

func cleanupCertificates() {
	os.Remove(testCAFile)
	os.Remove(testCertFile)
	os.Remove(testKeyFile)
}
