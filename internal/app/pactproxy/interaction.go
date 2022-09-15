package pactproxy

import (
	"encoding/json"
	"fmt"
	"mime"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	log "github.com/sirupsen/logrus"

	"github.com/PaesslerAG/jsonpath"
	"github.com/pkg/errors"
)

const (
	mediaTypeJSON = "application/json"
	mediaTypeText = "text/plain"
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
	Modifiers    *interactionModifiers
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

	matchingRules := getMatchingRules(request)
	regexString, err := getPathRegex(matchingRules)
	if err != nil {
		return nil, err
	}

	if regexString != "" {
		regex, err := regexp.Compile("^" + regexString + "$")

		if err != nil {
			return nil, errors.Wrap(err, "unable to parse interaction definition, cannot parse path regex rule")
		}

		matcher = &regexPathMatcher{val: regex}
	}
	propertiesWithMatchingRule := getBodyPropertiesWithMatchingRules(matchingRules)

	interaction := &interaction{
		pathMatcher: matcher,
		method:      request["method"].(string),
		Alias:       alias,
		definition:  definition,
		Description: description,
	}

	interaction.Modifiers = &interactionModifiers{
		interaction: interaction,
		modifiers:   sync.Map{},
	}

	requestBody, ok := request["body"]
	if !ok {
		return interaction, nil
	}

	mediaType, err := parseMediaType(request)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse media type")
	}

	switch mediaType {
	case mediaTypeJSON:
		if jsonRequestBody, ok := requestBody.(map[string]interface{}); ok {
			interaction.addJSONConstraintsFromPact("$.body", propertiesWithMatchingRule, jsonRequestBody)
			return interaction, nil
		}
		return nil, fmt.Errorf("media type is %s but body is not json", mediaType)
	case mediaTypeText:
		if plainTextRequestBody, ok := requestBody.(string); ok {
			interaction.addTextConstraintsFromPact(propertiesWithMatchingRule, plainTextRequestBody)
			return interaction, nil
		}
		return nil, fmt.Errorf("media type is %s but body is not text", mediaType)
	}
	return nil, fmt.Errorf("unsupported media type %s", mediaType)
}

// looks for a matching rule for key "$.path" in the supplied map
// if the found element is a map, it is treated as a pacs v2 style matching rule (i.e. "$.path": { "regex": "<expression>" } )
// if the found element is an array, it is treated as a pacs v3 list of matchers (i.e. "path": { "matchers": [ {"match": "regex", "regex": "<exp>"}]} )
func getPathRegex(matchingRules map[string]interface{}) (string, error) {
	var regexString string

	if pathRule, hasPathRule := matchingRules["$.path"]; hasPathRule {

		switch typedPathRule := pathRule.(type) {
		case map[string]interface{}:
			r, ok := typedPathRule["regex"].(string)
			if ok {
				regexString = r
			}
		case []interface{}:
			{
				for _, r := range typedPathRule {
					if m, ok := r.(map[string]interface{}); ok {
						if m["match"] == "regex" {
							if r, b := m["regex"].(string); b {
								regexString = r
							}
						}
					}
				}
			}
		}
	}
	// no path rule present
	return regexString, nil
}

func getMatchingRules(request map[string]interface{}) map[string]interface{} {
	rules, hasRules := request["matchingRules"]
	if !hasRules {
		rules = make(map[string]interface{})
	}
	rulesMap := rules.(map[string]interface{})
	return rulesMap
}

// Flattens a map with nested rules; stops when it finds a "matchers" or "regex" entry
/*
func flattenRulesMap(path string, mapToFlatten map[string]interface{}, outputMap map[string]interface{}) {
	for k, v := range mapToFlatten {
		k := strings.TrimPrefix(k, "$.")
		if k == "matchers" {
			outputMap[path] = v
		}
		if k == "regex" {
			outputMap[path] = map[string]interface{}{k: v}
		} else {
			switch val := v.(type) {
			case map[string]interface{}:
				flattenRulesMap(path+"."+k, val, outputMap)
			default:
				outputMap[path+"."+k] = v
			}
		}
	}
}
*/

// finds the paths of the body properties for which the matchingRules map
// contains matching rules
// It understands both v2 style matching rules (' "$.body.data.id": { "regex": "<exp>" } )
// and v3 style matching rules ( '"body": { "$.data.id": { "matchers": [...] } } } )
func getBodyPropertiesWithMatchingRules(matchingRules map[string]interface{}) map[string]bool {
	results := map[string]bool{}
	for k, v := range matchingRules {
		if strings.HasPrefix(k, "$.body") {
			// v2 style matchingRules
			results[k] = true
		} else if k == "body" {
			// this contains a map with the keys being the property names
			// and the values being the related matchers. We are only interested
			// in the property names here.
			if properties, ok := v.(map[string]interface{}); ok {
				for propertyname, _ := range properties {
					path := strings.TrimPrefix(propertyname, "$.")
					path = "$.body." + path
					results[path] = true
				}
			}
		}
	}
	return results
}

func parseMediaType(request map[string]interface{}) (string, error) {
	headers, hasHeaders := request["headers"]
	if !hasHeaders {
		log.Info("Request has no headers defined - defaulting media type to text/plain")
		return mediaTypeText, nil
	}

	parsed, ok := headers.(map[string]interface{})
	if !ok {
		return "", errors.New("incorrect format of request headers")
	}

	contentType, ok := parsed["Content-Type"]
	if !ok {
		log.Info("Request has no Content-Type header defined - defaulting media type to text/plain")
		return mediaTypeText, nil
	}

	contentTypeStr, ok := contentType.(string)
	if !ok {
		return "", errors.New("incorrect format of Content-Type header")
	}

	mediaType, _, err := mime.ParseMediaType(contentTypeStr)
	if err != nil {
		return "", err
	}

	return mediaType, nil
}

// This function adds constraints for all the fields in the JSON request body which do not
// have a corresponding matching rule
func (i *interaction) addJSONConstraintsFromPact(path string, matchingRules map[string]bool, values map[string]interface{}) {
	for k, v := range values {
		switch val := v.(type) {
		case map[string]interface{}:
			if _, exists := val["json_class"]; exists {
				continue
			}
			i.addJSONConstraintsFromPact(path+"."+k, matchingRules, val)
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

// This function adds constraints for the entire plain text request body if
// it doesn't have a corresponding matching rule
func (i *interaction) addTextConstraintsFromPact(matchingRules map[string]bool, constraint string) {
	if _, present := matchingRules["$.body"]; !present {
		i.AddConstraint(interactionConstraint{
			Path:   "$.body",
			Format: "%v",
			Values: []interface{}{constraint},
		})
	}
}
func (i *interaction) Match(path, method string) bool {
	return method == i.method && i.pathMatcher.match(path)
}

func (i *interaction) AddConstraint(constraint interactionConstraint) {
	i.constraints.Store(constraint.Key(), constraint)
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
