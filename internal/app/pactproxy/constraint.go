package pactproxy

import (
	"strings"
)

type interactionConstraint struct {
	Interaction string        `json:"interaction"`
	Path        string        `json:"path"`
	Values      []interface{} `json:"values"`
	Format      string        `json:"format"`
	Source      string        `json:"source"`
}

func (i interactionConstraint) Key() string {
	return strings.Join([]string{i.Interaction, i.Path}, "_")
}
