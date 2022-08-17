package pactproxy

import (
	"fmt"
	"reflect"
	"testing"
)

func TestLoadInteractionTextConstraints(t *testing.T) {
	matchersNotPresentJSON := `{
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

	matchersPresentJSON := `{
		"description": "Create report successfully",
		"providerState": [
		  {
			"name": "User has permission to create report",
			"params": {
			  "read": "reports",
			  "create_approve": "reports",
			  "organisation_id": "743d5b63-8e6f-432e-a8fa-c5d8d2ee5fcb",
			  "user_id": "b684c0b7-3375-4165-872b-19e9c21b903c"
			}
		  }
		],
		"request": {
		  "method": "POST",
		  "path": "/v1/reports",
		  "headers": {
			"Accept": "application/json; charset=utf-8",
			"Content-Type": "application/json; charset=utf-8",
			"X-Consumer-Custom-Id": "772ba46b-3053-49bf-9444-6c0784af1846"
		  },
		  "body": "some random file content",
		  "matchingRules": {
			"$.headers.Content-Type": {
			  "regex": "application\/json"
			},
			"$.body": {
			  "match": "type"
			}}
		},
		"response": {
		  "status": 201,
		  "headers": {
			"Content-Type": "application/json; charset=utf-8"
		  },
		  "body": {
			"data": {
			  "version": 0
			}
		  }
		}
	  }
	`
	type args struct {
		data  []byte
		alias string
	}
	tests := []struct {
		name           string
		args           args
		wantConstraint interactionConstraint
		wantErr        bool
	}{
		{
			name: "matchers not present in pact JSON",
			args: args{
				data:  []byte(matchersNotPresentJSON),
				alias: "matchers not present",
			},
			wantConstraint: interactionConstraint{
				Path:   "$.body",
				Format: "%v",
				Values: []interface{}{"some file request"},
			},
		},
		{
			name: "matchers present in pact JSON",
			args: args{
				data:  []byte(matchersPresentJSON),
				alias: "matchers not present",
			},
			wantConstraint: interactionConstraint{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadInteraction(tt.args.data, tt.args.alias)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadInteraction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			var gotConstraint interactionConstraint
			var present bool
			got.constraints.Range(func(key, value interface{}) bool {
				fmt.Printf("key %v value %v\n", key, value)
				gotConstraint, present = value.(interactionConstraint)
				return present
			})
			if !reflect.DeepEqual(gotConstraint, tt.wantConstraint) {
				t.Errorf("LoadInteraction() = %v, want %v", got, tt.wantConstraint)
			}
		})
	}
}
