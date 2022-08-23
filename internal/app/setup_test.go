package app

import (
	"bytes"
	"fmt"
	"github.com/pact-foundation/pact-go/dsl"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/form3tech-oss/pact-proxy/internal/app/configuration"
	"github.com/pact-foundation/pact-go/utils"
)

var (
	pathOnce               sync.Once
	adminURL               *url.URL
	proxyURL               *url.URL
	pact                   *dsl.Pact
	originalPactServerPort int
)

//TODO move these private funcs under Main..
func getTopLevelDir() (string, error) {
	gitCommand := exec.Command("git", "rev-parse", "--show-toplevel")
	var out bytes.Buffer
	gitCommand.Stdout = &out
	if err := gitCommand.Run(); err != nil {
		return "", err
	}

	topLevelDir := strings.TrimRight(out.String(), "\n")
	return topLevelDir, nil
}

func setPathOnce() {
	pathOnce.Do(func() {
		topLevelDir, err := getTopLevelDir()
		if err != nil {
			panic(err)
		}
		pactPath := filepath.Join(topLevelDir, "pact/bin")
		os.Setenv("PATH", pactPath+":"+os.Getenv("PATH"))
	})
}

func startPactServer(overridePort int) func() *dsl.Pact {
	pact = &dsl.Pact{
		Consumer: "MyConsumer",
		Provider: "MyProvider",
		Host:     "localhost",
	}

	pact.Setup(true)
	originalPactServerPort = pact.Server.Port
	pact.Server.Port = overridePort

	return pact.Teardown
}

func TestMain(m *testing.M) {
	setPathOnce()
	adminPort, err := utils.GetFreePort()
	if err != nil {
		panic(err)
	}

	adminServer := configuration.ServeAdminAPI(adminPort)
	defer adminServer.Close()

	adminURL, err = url.Parse(fmt.Sprintf("http://localhost:%d", adminPort))
	if err != nil {
		panic(err)
	}

	proxyPort, err := utils.GetFreePort()
	if err != nil {
		panic(err)
	}

	proxyURL, err = url.Parse(fmt.Sprintf("http://localhost:%d", proxyPort))
	if err != nil {
		panic(err)
	}

	teardown := startPactServer(proxyPort)
	defer teardown()

	m.Run()
}
