package pactproxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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
	_, err := p.client.Do(r)
	if err != nil {
		panic(err)
	}
}

func (p *PactProxy) addModifier(interaction, path, value string, attempt *int) {
	body := map[string]interface{}{
		"interaction": interaction,
		"path":        path,
		"value":       value,
	}
	if attempt != nil {
		body["attempt"] = attempt
	}
	b, _ := json.Marshal(body)

	r, _ := http.NewRequest("POST", strings.TrimSuffix(p.url, "/")+"/interactions/modifiers", bytes.NewBuffer(b))
	_, err := p.client.Do(r)
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

func (s InteractionSetup) AddConstraint(path, value string) InteractionSetup {
	s.pactProxy.addConstraint(s.interaction, path, value)
	return s
}

func (s InteractionSetup) AddModifier(path, value string, attempt *int) InteractionSetup {
	s.pactProxy.addModifier(s.interaction, path, value, attempt)
	return s
}

func (s InteractionSetup) AddConstraintFrom(path, fromInteraction, format string, values ...string) {
	s.pactProxy.addConstraintFrom(s.interaction, path, fromInteraction, format, values)
}
