package pactproxy

import (
	"encoding/json"
	"fmt"
	"github.com/tidwall/sjson"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type interactionModifier struct {
	Interaction string `json:"interaction"`
	Path        string `json:"path"`
	Value       string `json:"value"`
	Attempt     *int   `json:"attempt"`
	countStatusCode       int
	countBody       int
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

func (i *interactionModifier) modifyBody(b []byte) ([]byte, error)  {
	if len(i.Path) < 7 {
		return nil, fmt.Errorf("invalid path: %s", i.Path)
	}
	if i.Path[:7] == "$.body." {
		i.countBody++
		if i.Attempt == nil || *i.Attempt == i.countBody {
			out, err := sjson.SetBytes(b, i.Path[7:], i.Value)
			return out, err
		}
	}
	return b, nil
}

func (i *interactionModifier) modifyStatusCode() (bool, int) {
	if i.Path == "$.status" {
		i.countStatusCode++
		if i.Attempt == nil || *i.Attempt == i.countStatusCode {
			code, err := strconv.Atoi(i.Value)
			if err == nil {
				return true, code
			}
		}
	}
	return false, 0
}
