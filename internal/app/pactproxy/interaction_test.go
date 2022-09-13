package pactproxy

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLoadInteractionPlainTextConstraints(t *testing.T) {
	matchersNotPresent := `{
		"description": "A request to create an address",
		"request": {
		  "method": "POST",
		  "path": "/addresses",
		  "headers": {
			"Content-Type": "text/plain"
		  },
		  "body": "some file request"
		},
		"response": {
		  "status": 200,
		  "headers": {
			"Content-Type": "text/plain"
		  },
		  "body": "some file response"
		}
	  }`
	matcherPresent :=
		`{
		"description": "A request to create an address",
		"request": {
		  "method": "POST",
		  "path": "/addresses",
		  "headers": {
			"Content-Type": "text/plain"
		  },
		  "body": "some file request",
		  "matchingRules": {
			"$.body": {
			  "regex": "type"
			}}
		},
		"response": {
		  "status": 200,
		  "headers": {
			"Content-Type": "text/plain"
		  },
		  "body": "some file response"
		}
	  }`
	invalidBodyPathMatcherPresent :=
		`{
		"description": "A request to create an address",
		"request": {
		  "method": "POST",
		  "path": "/addresses",
		  "headers": {
			"Content-Type": "text/plain"
		  },
		  "body": "some file request",
		  "matchingRules": {
			"$.body.invalid.path": {
			  "regex": "type"
			}}
		},
		"response": {
		  "status": 200,
		  "headers": {
			"Content-Type": "text/plain"
		  },
		  "body": "some file response"
		}
	  }`
	tests := []struct {
		name           string
		interaction    []byte
		wantConstraint interactionConstraint
		wantErr        bool
	}{
		{
			name:        "matcher not present - interaction is created",
			interaction: []byte(matchersNotPresent),
			wantConstraint: interactionConstraint{
				Path:   "$.body",
				Format: "%v",
				Values: []interface{}{"some file request"},
			},
		},
		{
			name:        "matcher with invalid path present - interaction is created",
			interaction: []byte(invalidBodyPathMatcherPresent),
			wantConstraint: interactionConstraint{
				Path:   "$.body",
				Format: "%v",
				Values: []interface{}{"some file request"},
			},
		},
		{
			name:           "matcher present - no interaction created",
			interaction:    []byte(matcherPresent),
			wantConstraint: interactionConstraint{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadInteraction(tt.interaction, "alias")

			require.Equalf(t, tt.wantErr, err != nil, "error %v", err)

			var gotConstraint interactionConstraint
			got.constraints.Range(func(key, value interface{}) bool {
				var present bool
				gotConstraint, present = value.(interactionConstraint)
				return present
			})

			assert.EqualValues(t, tt.wantConstraint, gotConstraint)
		})
	}
}

// This test asserts that given a pact v3-style nested matching rule, a constraint
// is not created for the corresponding property
func TestV3MatchingRulesLeadToCorrectConstraints(t *testing.T) {
	v3matchingRulePresent :=
		`{
		"description": "A request to admit a payment",
		"request": {
		  "method": "POST",
		  "path": "/v1/payments/830e5d93-1cd1-4def-953e-6188d7235c38/admissions",
		  "headers": {
			"Content-Type": "application/json; charset=utf-8"
		  },
		  "body": {
		    "data": {
		      "type": "payment_admissions"
		    }
		  },
		  "matchingRules": {
			"body": {
              "$.data.type" :{
                "matchers" : [
                  { "match": "regex", "regex": "[a-zA-z]*" }
                ]
              }
		    }
		  }
		},
		"response": {
		  "status": 200,
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "body": {
		    "data": {
		      "type": "payment_admissions"
		    }
		  }
		}
	  }`
	tests := []struct {
		name           string
		interaction    []byte
		wantConstraint interactionConstraint
		wantErr        bool
	}{
		{
			name:           "v3 matching rule present - no constraint created for body.data.type",
			interaction:    []byte(v3matchingRulePresent),
			wantConstraint: interactionConstraint{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadInteraction(tt.interaction, "alias")

			require.Equalf(t, tt.wantErr, err != nil, "error %v", err)

			var gotConstraint interactionConstraint
			got.constraints.Range(func(key, value interface{}) bool {
				var present bool
				gotConstraint, present = value.(interactionConstraint)
				return present
			})

			assert.EqualValues(t, tt.wantConstraint, gotConstraint)
		})
	}
}

func Test_parseMediaType(t *testing.T) {
	tests := []struct {
		name    string
		request map[string]interface{}
		want    string
		wantErr bool
	}{
		{
			name: "valid media type",
			request: map[string]interface{}{
				"headers": map[string]interface{}{
					"Content-Type": "application/json",
				}},
			want: "application/json",
		},
		{
			name: "valid media type with encoding",
			request: map[string]interface{}{
				"headers": map[string]interface{}{
					"Content-Type": "application/json; charset=utf-8",
				}},
			want: "application/json",
		},
		{
			name:    "request headers definition missing - default to text",
			request: map[string]interface{}{},
			want:    "text/plain",
		},
		{
			name: "request Content-Type header missing - default to text",
			request: map[string]interface{}{
				"headers": map[string]interface{}{
					"other header": "stuff",
				}},
			want: "text/plain",
		},
		{
			name: "invalid media type",
			request: map[string]interface{}{
				"headers": map[string]interface{}{
					"Content-Type": "invalid/value/contentType",
				}},
			wantErr: true,
		},
		{
			name: "empty Content-Type header",
			request: map[string]interface{}{
				"headers": map[string]interface{}{
					"Content-Type": "",
				}},
			wantErr: true,
		},
		{
			name: "request headers is not map type",
			request: map[string]interface{}{
				"headers": "string"},
			wantErr: true,
		},
		{
			name: "Content-Type header is not string type",
			request: map[string]interface{}{
				"headers": map[string]interface{}{
					"Content-Type": []string{"slice"},
				}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMediaType(tt.request)

			require.Equalf(t, tt.wantErr, err != nil, "got error: %v", err)
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
