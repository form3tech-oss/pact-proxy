package pactproxy

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sync"
	"sync/atomic"

	log "github.com/sirupsen/logrus"

	"github.com/PaesslerAG/jsonpath"
	"github.com/pkg/errors"
)

type pathMatcher interface {
	match(val string) bool
}

type stringPathMatcher struct {
	val string
}

func (m *stringPathMatcher) match(val string) bool {
	return val == m.val
}

type regexPathMatcher struct {
	val *regexp.Regexp
}

func (m *regexPathMatcher) match(val string) bool {
	return m.val.MatchString(val)
}

type interaction struct {
	pathMatcher  pathMatcher
	method       string
	Alias        string
	Description  string
	definition   map[string]interface{}
	constraints  sync.Map
	modifiers    sync.Map
	lastRequest  atomic.Value
	requestCount int32
}

func LoadInteraction(data []byte, alias string) (*interaction, error) {
	definition := make(map[string]interface{})
	err := json.Unmarshal(data, &definition)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse interaction definition")
	}

	description, ok := definition["description"].(string)
	if !ok {
		return nil, errors.New("unable to parse interaction definition, no Description defined")
	}

	request, ok := definition["request"].(map[string]interface{})
	if !ok {
		return nil, errors.New("unable to parse interaction definition, no request defined")
	}

	var matcher pathMatcher = &stringPathMatcher{val: request["path"].(string)}

	matchingRules, hasRules := request["matchingRules"]
	if !hasRules {
		matchingRules = make(map[string]interface{})
	}

	if pathRule, hasPathRule := matchingRules.(map[string]interface{})["$.path"]; hasPathRule {
		regexRule := pathRule.(map[string]interface{})["regex"].(string)
		regex, err := regexp.Compile("^" + regexRule + "$")
		if err != nil {
			return nil, errors.Wrap(err, "unable to parse interaction definition, cannot parse path regex rule")
		}

		matcher = &regexPathMatcher{val: regex}
	}

	interaction := &interaction{
		pathMatcher: matcher,
		method:      request["method"].(string),
		Alias:       alias,
		definition:  definition,
		Description: description,
	}

	if requestBody, hasRequestBody := request["body"]; hasRequestBody {
		interaction.addConstraintsFromPact("$.body", matchingRules.(map[string]interface{}), requestBody.(map[string]interface{}))
	}

	if !hasRules {
		matchingRules = make(map[string]interface{})
	}

	return interaction, nil
}

func (i *interaction) addConstraintsFromPact(path string, matchingRules, values map[string]interface{}) {
	for k, v := range values {
		switch val := v.(type) {
		case map[string]interface{}:
			if _, exists := val["json_class"]; exists {
				continue
			}
			i.addConstraintsFromPact(path+"."+k, matchingRules, val)
		default:
			p := path + "." + k
			if _, hasRule := matchingRules[p]; !hasRule {
				i.AddConstraint(interactionConstraint{
					Path:   p,
					Format: "%v",
					Values: []interface{}{val},
				})
			}
		}
	}
}

func (i *interaction) Match(path, method string) bool {
	return method == i.method && i.pathMatcher.match(path)
}

func (i *interaction) AddConstraint(constraint interactionConstraint) {
	i.constraints.Store(constraint.Key(), constraint)
}

func (i *interaction) AddModifier(modifier interactionModifier) {
	i.modifiers.Store(modifier.Key(), modifier)
}

func (i *interaction) loadValuesFromSource(constraint interactionConstraint, interactions *Interactions) ([]interface{}, error) {
	values := append([]interface{}(nil), constraint.Values...)
	sourceInteraction, ok := interactions.Load(constraint.Source)
	if !ok {
		return nil, errors.Errorf("cannot find source interaction '%s' for constraint", constraint.Source)
	}

	sourceRequest, ok := sourceInteraction.lastRequest.Load().(requestDocument)
	if !ok {
		return nil, errors.Errorf("source interaction '%s' as no requests", constraint.Source)
	}

	for i, v := range constraint.Values {
		values[i], _ = jsonpath.Get(v.(string), map[string]interface{}(sourceRequest))
	}

	return values, nil
}

func (i *interaction) EvaluateConstrains(request requestDocument, interactions *Interactions) (bool, []string) {
	result := true
	violations := make([]string, 0)

	i.constraints.Range(func(_, v interface{}) bool {
		constraint := v.(interactionConstraint)
		values := constraint.Values
		if constraint.Source != "" {
			var err error
			values, err = i.loadValuesFromSource(constraint, interactions)
			if err != nil {
				violations = append(violations, err.Error())
				result = false
				return true
			}
		}

		actual := ""
		val, err := jsonpath.Get(request.encodeValues(constraint.Path), map[string]interface{}(request))
		if err != nil {
			log.Warn(err)
		}
		if reflect.TypeOf(val) == reflect.TypeOf([]interface{}{}) {
			log.Infof("skipping matching on interface{} type for path '%s'", constraint.Path)
			return true
		}
		if err == nil {
			actual = fmt.Sprintf("%v", val)
		}

		expected := fmt.Sprintf(constraint.Format, values...)
		if actual != expected {
			violations = append(violations, fmt.Sprintf("value '%s' at path '%s' does not match constraint '%s'", actual, constraint.Path, expected))
			result = false
		}

		return true
	})

	return result, violations
}

func (i *interaction) StoreRequest(request requestDocument) {
	i.lastRequest.Store(request)
	atomic.AddInt32(&i.requestCount, 1)
}

func (i *interaction) HasRequests(count int) bool {
	return atomic.LoadInt32(&i.requestCount) >= int32(count)
}
