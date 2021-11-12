package pactproxy

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type interactionModifier struct {
	Interaction string `json:"interaction"`
	Path        string `json:"path"`
	Value       string `json:"value"`
	Attempt     *int   `json:"attempt"`
	count       int
}

func loadModifier(data []byte) (*interactionModifier, error) {
	modifier := &interactionModifier{}
	err := json.Unmarshal(data, &modifier)
	if err != nil {
		return modifier, errors.Wrap(err, "unable to parse interactionModifier from data")
	}
	return modifier, nil
}

func (i *interactionModifier) Key() string {
	return strings.Join([]string{i.Interaction, i.Path}, "_")
}

func (i *interactionModifier) modifyStatusCode(res http.ResponseWriter) bool {
	if i.Path == "$.status" {
		i.count++
		if i.Attempt == nil || *i.Attempt == i.count {
			code, err := strconv.Atoi(i.Value)
			if err == nil {
				res.WriteHeader(code)
				return true
			}
		}
	}
	return false
}
