package pactproxy

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			interaction, err := LoadInteraction(tt.interaction, "alias")

			require.Equalf(t, tt.wantErr, err != nil, "error %v", err)

			var foundConstraint interactionConstraint
			for _, constraint := range interaction.constraints {
				foundConstraint = constraint
				break
			}

			assert.EqualValues(t, tt.wantConstraint, foundConstraint)
		})
	}
}

func TestLoadInteractionJSONConstraints(t *testing.T) {
	arrMatchersNotPresent := `{
		"description": "A request to create an address",
		"request": {
		  "method": "POST",
		  "path": "/addresses",
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "body": {
			"addressLines": ["line 1", "line 2"]
		  }
		},
		"response": {
		  "status": 200,
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "body": {
			"addressLines": ["line 1", "line 2"]
		  }
		}
	  }`
	arrMatcherPresent :=
		`{
		"description": "A request to create an address",
		"request": {
		  "method": "POST",
		  "path": "/addresses",
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "body": {
			"addressLines": ["line 1", "line 2"]
		  },
		  "matchingRules": {
			"$.body.addressLines[0]": {
			  "regex": ".*"
			}
		  }
		},
		"response": {
		  "status": 200,
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "body": {
			"addressLines": ["line 1", "line 2"]
		  }
		}
	  }`
	tests := []struct {
		name            string
		interaction     []byte
		wantConstraints []interactionConstraint
	}{
		{
			name:        "array and matcher not present - interactions are created per element",
			interaction: []byte(arrMatchersNotPresent),
			wantConstraints: []interactionConstraint{
				{
					Path:   "$.body.addressLines[0]",
					Format: "%v",
					Values: []interface{}{"line 1"},
				},
				{
					Path:   "$.body.addressLines[1]",
					Format: "%v",
					Values: []interface{}{"line 2"},
				},
				{
					Path:   "$.body.addressLines",
					Format: fmtLen,
					Values: []interface{}{2},
				},
			},
		},
		{
			name:        "array and matcher present - interaction is not created for matched element",
			interaction: []byte(arrMatcherPresent),
			wantConstraints: []interactionConstraint{
				{
					Path:   "$.body.addressLines[1]",
					Format: "%v",
					Values: []interface{}{"line 2"},
				},
				{
					Path:   "$.body.addressLines",
					Format: fmtLen,
					Values: []interface{}{2},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interaction, err := LoadInteraction(tt.interaction, "alias")
			require.NoError(t, err)

			actual := make([]interactionConstraint, 0, len(interaction.constraints))
			for _, constraint := range interaction.constraints {
				actual = append(actual, constraint)
			}
			assert.ElementsMatch(t, tt.wantConstraints, actual)
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
            "path": {
              "matchers" : [
                { "match": "regex", "regex": "/v1/payments/[0-9A-Fa-f]{8}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{12}/admissions" }
              ]
            },
			"body": {
              "$.data.type" :{
                "matchers" : [
                  { "match": "regex", "regex": ".*" }
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

	multiplev3matchingRulesPresent :=
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
		      "type": "payment_admissions",
              "attributes": {
                "status": "failed",
                "status_reason": "unknown_accountnumber"
			  }
		    }
		  },
		  "matchingRules": {
            "path": {
              "matchers" : [
                { "match": "regex", "regex": "/v1/payments/[0-9A-Fa-f]{8}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{4}[-][0-9A-Fa-f]{12}/admissions" }
              ]
            },
			"body": {
              "$.data.type" :{
                "matchers" : [
                  { "match": "regex", "regex": ".*" }
                ]
			  },
              "$.data.attributes.status_reason" :{
			    "matchers" : [
			      { "match": "regex", "regex": "(unknown_accountnumber|account_closed|invalid_beneficiary_details)" }
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
		{
			name:        "multiple v3 matching rules present - no constraint created for body properties with matching rule",
			interaction: []byte(multiplev3matchingRulesPresent),
			wantConstraint: interactionConstraint{
				Path:   "$.body.data.attributes.status",
				Format: "%v",
				Values: []interface{}{"failed"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interaction, err := LoadInteraction(tt.interaction, "alias")

			require.Equalf(t, tt.wantErr, err != nil, "error %v", err)

			var foundConstraint interactionConstraint
			for _, constraint := range interaction.constraints {
				foundConstraint = constraint
				break
			}

			assert.EqualValues(t, tt.wantConstraint, foundConstraint)
		})
	}
}

func TestLoadArrayRequestBodyInteractions(t *testing.T) {
	arrayOfStrings := `{
		"description": "A request to create an address",
		"request": {
		  "method": "POST",
		  "path": "/addresses",
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "body": ["a", "b", "c"]
		},
		"response": {
		  "status": 200,
		  "headers": {
			"Content-Type": "application/json"
		  },
		  "body": ["a", "b", "c"]
		}
	  }`
	arrayOfInts := `{
		"description": "A request to create an address",
		"request": {
			"method": "POST",
			"path": "/addresses",
			"headers": {
			"Content-Type": "application/json"
			},
			"body": [1, 2, 3]
		},
		"response": {
			"status": 200,
			"headers": {
			"Content-Type": "application/json"
			},
			"body": [1, 2, 3]
		}
		}`
	arrayOfBools := `{
			"description": "A request to create an address",
			"request": {
				"method": "POST",
				"path": "/addresses",
				"headers": {
				"Content-Type": "application/json"
				},
				"body": [true, false, true]
			},
			"response": {
				"status": 200,
				"headers": {
				"Content-Type": "application/json"
				},
				"body": [true, false, true]
			}
			}`
	arrayOfObjects := `{
			"description": "A request to create an address",
			"request": {
				"method": "POST",
				"path": "/addresses",
				"headers": {
				"Content-Type": "application/json"
				},
				"body": [ {"key": "val"}, {"key": "val"} ]
			},
			"response": {
				"status": 200,
				"headers": {
				"Content-Type": "application/json"
				},
				"body": [ {"key": "val"}, {"key": "val"} ]
			}
			}`
	arrayOfStringsWithMatcher :=
		`{
		"description": "A request to create an address",
		"request": {
		  "method": "POST",
		  "path": "/addresses",
			"headers": {
			"Content-Type": "application/json"
		  },
		  "body": ["a", "b", "c"],
		  "matchingRules": {
				"$.body": {
					"match": "type"
				}
			}
		},
		"response": {
		  "status": 200,
		  "headers": {
			"Content-Type": "application/json"
			},
			"body": ["a", "b", "c"]
		}
	  }`

	tests := []struct {
		name        string
		interaction []byte
	}{
		{
			name:        "array of strings",
			interaction: []byte(arrayOfStrings),
		},
		{
			name:        "array of ints",
			interaction: []byte(arrayOfInts),
		},
		{
			name:        "array of bools",
			interaction: []byte(arrayOfBools),
		},
		{
			name:        "array of objects",
			interaction: []byte(arrayOfObjects),
		},
		{
			name:        "array of strings with matcher",
			interaction: []byte(arrayOfStringsWithMatcher),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interaction, err := LoadInteraction(tt.interaction, "alias")
			require.NoError(t, err, "unexpected error %v", err)

			require.Empty(t, interaction.constraints, "No constraint should be added for the interaction")
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

func Test_getPathRegex(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		want    string
		wantErr bool
	}{
		{
			"v2 pact matching rules",
			`{"$.path":{ "regex": "1234"}}`,
			"1234",
			false,
		},
		{
			"v2 pact matching rules invalid content",
			`{"$.path":{ "invalid": "1234"}}`,
			"",
			true,
		},
		{
			"v3 pact matching rules - valid ",
			`{"path":{ "matchers": [{
								"match": "regex",
								"regex": "1234"
							  }]}}`,

			"1234",
			false,
		},
		{
			"v3 pact matching rules - invalid match type",
			`{"path":{ "matchers": [{
								"match": "invalid",
								"regex": "1234"
							  }]}}`,

			"",
			true,
		},
		{
			"v3 pact matching rules - multiple valid",
			`{"path":{ "matchers": [
							{
								"match": "test"
							},
							{
								"match": "regex",
								"regex": "1234"
							}, 
							{
								"match": "type"
							}
							]}}`,
			"1234",
			false,
		},
		{
			"v3 pact matching rules - invalid content",
			`{"path":{ "invalid": [{
								"match": "regex",
								"regex": "1234"
							  }]}}`,
			"",
			true,
		},
		{
			"v3 pact matching rules - regex field is not found",
			`{"path":{ "invalid": [{
								"match": "regex" }]}}`,
			"",
			true,
		},
		{
			"v3 pact matching rules - invalid match key",
			`{"path":{ "matchers": [{
								"match": "regex",
								"invalid": "1234"
							  }]
					}}`,
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string]interface{}{}
			err := json.Unmarshal([]byte(tt.args), &input)
			assert.NoError(t, err)
			got, err := getPathRegex(input)
			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equalf(t, tt.want, got, "getPathRegex(%v)", tt.args)
		})
	}
}
