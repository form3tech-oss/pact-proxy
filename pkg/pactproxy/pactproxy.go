package pactproxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type PactProxy struct {
	client http.Client
	url    string
}

type InteractionSetup struct {
	interaction string
	pactProxy   *PactProxy
}

func New(url string) *PactProxy {
	return &PactProxy{
		client: http.Client{
			Timeout: 30 * time.Second,
		},
		url: url,
	}
}

func (p *PactProxy) ForInteraction(interaction string) *InteractionSetup {
	return &InteractionSetup{
		interaction: interaction,
		pactProxy:   p,
	}
}

func (p *PactProxy) addConstraint(interaction, pactPath, value string) {
	b, _ := json.Marshal(map[string]interface{}{
		"interaction": interaction,
		"path":        pactPath,
		"format":      "%s",
		"values":      []string{value},
	})

	r, _ := http.NewRequest("POST", strings.TrimSuffix(p.url, "/")+"/interactions/constraints", bytes.NewBuffer(b))
	r.Header.Set("Content-Type", "application/json")
	_, err := p.client.Do(r)
	if err != nil {
		panic(err)
	}
}

func (p *PactProxy) addModifier(interaction, path string, value interface{}, attempt *int) {
	body := map[string]interface{}{
		"interaction": interaction,
		"path":        path,
		"value":       value,
	}
	if attempt != nil {
		body["attempt"] = attempt
	}
	b, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}

	r, _ := http.NewRequest("POST", strings.TrimSuffix(p.url, "/")+"/interactions/modifiers", bytes.NewBuffer(b))
	r.Header.Set("Content-Type", "application/json")
	_, err = p.client.Do(r)
	if err != nil {
		panic(err)
	}
}

func (p *PactProxy) addConstraintFrom(interaction, pactPath, fromInteraction, format string, values []string) {
	b, err := json.Marshal(map[string]interface{}{
		"interaction": interaction,
		"path":        pactPath,
		"source":      fromInteraction,
		"format":      format,
		"values":      values,
	})
	if err != nil {
		panic(err)
	}

	r, err := http.NewRequest("POST", strings.TrimSuffix(p.url, "/")+"/interactions/constraints", bytes.NewBuffer(b))
	if err != nil {
		log.Warn(err.Error())
		return
	}

	r.Header.Set("Content-Type", "application/json")
	res, err := p.client.Do(r)
	if err != nil {
		log.Warn(err.Error())
		return
	}

	if res.StatusCode != http.StatusOK {
		log.Warnf("failed to add constraint. %d", res.StatusCode)
	}
}

func (p *PactProxy) WaitForAll() error {
	r, _ := http.NewRequest("GET", strings.TrimSuffix(p.url, "/")+"/interactions/wait", nil)
	res, err := p.client.Do(r)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("timout waiting for interactions")
	}
	return nil
}

func (p *PactProxy) WaitForInteraction(interaction string, count int) error {
	q := url.Values{}
	q.Add("interaction", interaction)
	q.Add("count", strconv.Itoa(count))

	r, _ := http.NewRequest("GET", strings.TrimSuffix(p.url, "/")+"/interactions/wait?"+q.Encode(), nil)
	res, err := p.client.Do(r)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("fail")
	}
	return nil
}

func (p *PactProxy) ReadInteractionDetails(alias string) (*Interaction, error) {
	url := fmt.Sprintf("%s/interactions/details/%s", strings.TrimSuffix(p.url, "/"), alias)
	r, _ := http.NewRequest("GET", url, nil)
	res, err := p.client.Do(r)
	if err != nil {
		return nil, errors.Wrap(err, "http get")
	}
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("fail")
	}
	interaction := &Interaction{}
	err = json.NewDecoder(res.Body).Decode(interaction)
	if err != nil {
		return nil, err
	}

	return interaction, nil
}

func (p *PactProxy) IsReady() error {
	res, err := p.client.Get(strings.TrimSuffix(p.url, "/") + "/ready")
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected status code" + strconv.Itoa(res.StatusCode))
	}
	return nil
}

func (s InteractionSetup) AddConstraint(path, value string) InteractionSetup {
	s.pactProxy.addConstraint(s.interaction, path, value)
	return s
}

// AddModifier modifies the interaction's response on the specified attempt.
// Attempts start and index 1, and consider the count of _all_ executions since the interaction was registered, not
// since the modification was added.
func (s InteractionSetup) AddModifier(path string, value interface{}, attempt *int) InteractionSetup {
	s.pactProxy.addModifier(s.interaction, path, value, attempt)
	return s
}

func (s InteractionSetup) AddConstraintFrom(path, fromInteraction, format string, values ...string) {
	s.pactProxy.addConstraintFrom(s.interaction, path, fromInteraction, format, values)
}
