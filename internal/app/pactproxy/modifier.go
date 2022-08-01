package pactproxy

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
)

type interactionModifier struct {
	Interaction string `json:"interaction"`
	Path        string `json:"path"`
	Value       string `json:"value"`
	Attempt     *int   `json:"attempt"`
}

type interactionModifiers struct {
	interaction *interaction
	modifiers   sync.Map
}

func loadModifier(data []byte) (*interactionModifier, error) {
	modifier := &interactionModifier{}
	err := json.Unmarshal(data, &modifier)
	if err != nil {
		return modifier, errors.Wrap(err, "unable to parse interactionModifier from data")
	}
	return modifier, nil
}

func (im *interactionModifier) Key() string {
	return strings.Join([]string{im.Interaction, im.Path}, "_")
}

func (ims *interactionModifiers) AddModifier(modifier *interactionModifier) {
	ims.modifiers.Store(modifier.Key(), modifier)
}

func (ims *interactionModifiers) Modifiers() []*interactionModifier {
	var result []*interactionModifier
	ims.modifiers.Range(func(_, v interface{}) bool {
		result = append(result, v.(*interactionModifier))
		return true
	})
	return result
}

func (ims *interactionModifiers) modifyBody(b []byte) ([]byte, error) {
	for _, m := range ims.Modifiers() {
		if strings.HasPrefix(m.Path, "$.body.") {
			if m.Attempt == nil || *m.Attempt == int(atomic.LoadInt32(&ims.interaction.requestCount)) {
				var err error
				b, err = sjson.SetBytes(b, m.Path[7:], m.Value)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return b, nil
}

func (ims *interactionModifiers) modifyStatusCode() (bool, int) {
	for _, m := range ims.Modifiers() {
		if m.Path == "$.status" {
			if m.Attempt == nil || *m.Attempt == int(atomic.LoadInt32(&ims.interaction.requestCount)) {
				code, err := strconv.Atoi(m.Value)
				if err == nil {
					return true, code
				}
			}
		}
	}
	return false, 0
}
