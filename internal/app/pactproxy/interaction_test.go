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
