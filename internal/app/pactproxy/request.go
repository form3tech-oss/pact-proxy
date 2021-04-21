package pactproxy

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

type requestDocument map[string]interface{}

func LoadRequest(data []byte, url *url.URL) (requestDocument, error) {
	body := make(map[string]interface{})
	if len(data) > 0 {
		err := json.Unmarshal(data, &body)
		if err != nil {
			return nil, errors.Wrap(err, "unable to parse requestDocument body")
		}
	}

	queryValues := make(map[string]interface{})
	for q, v := range url.Query() {
		if len(v) > 0 {
			escapeValue(queryValues, q, v[0])
		}
	}

	return map[string]interface{}{
		"path":  url.Path,
		"body":  body,
		"query": queryValues,
	}, nil
}

func (r requestDocument) encodeValues(val string) string {
	query := r["query"].(map[string]interface{})
	return encodeMapValues(query, val)
}

func encodeMapValues(m map[string]interface{}, val string) string {
	result := val
	for k, v := range m {
		result = strings.ReplaceAll(result, "["+k+"]", "[\""+k+"\"]")
		switch val := v.(type) {
		case map[string]interface{}:
			result = encodeMapValues(val, result)
		}
	}
	return result
}

func escapeValue(values map[string]interface{}, query, val string) {
	open := strings.Index(query, "[")
	if open > -1 {
		key := query[:open]
		rest := query[open+1:]
		closing := strings.Index(rest, "]")
		if closing < 0 {
			values[query] = val
			return
		}

		subKey := rest[:closing]
		next := rest[closing+1:]

		existingValue := values[key]
		valueMap, ok := existingValue.(map[string]interface{})
		if !ok {
			valueMap = make(map[string]interface{})
			values[key] = valueMap
		}
		escapeValue(valueMap, subKey+next, val)
		return
	}
	values[query] = val
}
