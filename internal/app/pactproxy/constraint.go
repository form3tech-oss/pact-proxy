package pactproxy

import (
	"fmt"
	"strings"
)

const fmtLen = "_length_"

type interactionConstraint struct {
	Interaction string        `json:"interaction"`
	Path        string        `json:"path"`
	Values      []interface{} `json:"values"`
	Format      string        `json:"format"`
	Source      string        `json:"source"`
}

func (i interactionConstraint) Key() string {
	return strings.Join([]string{i.Interaction, i.Path}, "_")
}

func (i interactionConstraint) check(expectedValues []interface{}, actualValue interface{}) error {
	if i.Format == fmtLen {
		if len(expectedValues) != 1 {
			return fmt.Errorf(
				"expected single positive integer value for path %q length constraint, but there are %v expected values",
				i.Path, len(expectedValues))
		}
		expected, ok := expectedValues[0].(int)
		if !ok || expected < 0 {
			return fmt.Errorf("expected value for %q length constraint must be a positive integer", i.Path)
		}

		actualSlice, ok := actualValue.([]interface{})
		if !ok {
			return fmt.Errorf("value at path %q must be an array due to length constraint", i.Path)
		}
		if expected != len(actualSlice) {
			return fmt.Errorf("value of length %v at path %q does not match length constraint %v",
				expected, i.Path, len(actualSlice))
		}
		return nil
	}

	expected := fmt.Sprintf(i.Format, expectedValues...)
	actual := fmt.Sprintf("%v", actualValue)
	if expected != actual {
		return fmt.Errorf("value %q at path %q does not match constraint %q", actual, i.Path, expected)
	}
	return nil
}
