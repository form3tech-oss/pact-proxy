package app

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"

	"github.com/pact-foundation/pact-go/utils"
)

var (
	pathOnce               sync.Once
	proxyURL               *url.URL
	pact                   *dsl.Pact
	originalPactServerPort int
)

func TestMain(m *testing.M) {
	setPathOnce()

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
		Consumer:             "MyConsumer",
		Provider:             "MyProvider",
		Host:                 "localhost",
		SpecificationVersion: 3,
	}

	pact.Setup(true)
	originalPactServerPort = pact.Server.Port
	pact.Server.Port = overridePort

	return pact.Teardown
}
