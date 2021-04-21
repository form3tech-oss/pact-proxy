package pactproxy

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
)

type interactionConstraint struct {
	Interaction string        `json:"interaction"`
	Path        string        `json:"path"`
	Values      []interface{} `json:"values"`
	Format      string        `json:"format"`
	Source      string        `json:"source"`
}

func LoadConstraint(data []byte) (interactionConstraint, error) {
	constraint := interactionConstraint{}
	err := json.Unmarshal(data, &constraint)
	if err != nil {
		return constraint, errors.Wrap(err, "unable to parse interactionConstraint from data")
	}
	return constraint, nil
}

func (i interactionConstraint) Key() string {
	return strings.Join([]string{i.Interaction, i.Path}, "_")
}
