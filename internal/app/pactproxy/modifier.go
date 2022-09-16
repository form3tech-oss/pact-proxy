package pactproxy

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
)

type UnsupportedTypeSpecifiedError struct {
	receivedType string
}

func (e UnsupportedTypeSpecifiedError) Error() string {
	return fmt.Sprintf("Unsupported type specified: %s", e.receivedType)
}

type interactionModifier struct {
	Interaction string      `json:"interaction"`
	Path        string      `json:"path"`
	Value       interface{} `json:"value"`
	ValueType   string      `json:"type"`
	Attempt     *int        `json:"attempt"`
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
	valueAsString := fmt.Sprintf("%v", modifier.Value)
	switch modifier.ValueType {
	case "string":
		return modifier, nil
	case "int":
		intVal, err := strconv.Atoi(valueAsString)
		if err != nil {
			return nil, err
		}
		modifier.Value = intVal
	case "int32":
		intVal, err := strconv.ParseInt(valueAsString, 10, 32)
		if err != nil {
			return nil, err
		}
		modifier.Value = intVal
	case "int64":
		intVal, err := strconv.ParseInt(valueAsString, 10, 64)
		if err != nil {
			return nil, err
		}
		modifier.Value = intVal
	default:
		return nil, &UnsupportedTypeSpecifiedError{receivedType: fmt.Sprintf("%T", modifier.Value)}
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
				code, ok := m.Value.(int)
				if !ok {
					return false, 0
				}
				return true, code
			}
		}
	}
	return false, 0
}
