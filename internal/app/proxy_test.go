package app

import (
	"net/http"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
)

func TestLargePactResponse(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names_and_returns_large_body_and_any_name()

	when.
		a_request_is_sent_using_the_name("jane")

	then.
		pact_verification_is_successful().and().
		the_nth_response_body_has_(1, "large_string", largeString)
}

func TestLargePactResponseWithModifiedBody(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names_and_returns_large_body_and_any_name().and().
		a_modified_response_body_of_("$.body.name", "custom_name")

	when.
		a_request_is_sent_using_the_name("jane")

	then.
		pact_verification_is_successful().and().
		the_response_name_is_("custom_name").and()
}

func TestLargePactResponseWithModifiedStatusCode(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names_and_returns_large_body_and_any_name().and().
		a_modified_response_status_of_(http.StatusInternalServerError)

	when.
		a_request_is_sent_using_the_name("jane")

	then.
		pact_verification_is_successful().and().
		the_response_is_(http.StatusInternalServerError)
}

func TestConstraintMatches(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names().and().
		a_name_constraint_is_added("sam")

	when.
		a_request_is_sent_using_the_name("sam")

	then.
		pact_verification_is_successful().and().
		pact_can_be_generated()
}

func TestConstraintDoesntMatch(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names()

	when.
		a_name_constraint_is_added("sam").and().
		a_request_is_sent_using_the_name("bob")

	then.
		pact_verification_is_not_successful()
}

func TestConstraintHeaderMatch(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names().and().
		a_content_type_constraint_is_added("application/json")

	when.
		a_request_is_sent_using_the_name("sam")

	then.
		pact_verification_is_successful().and().
		pact_can_be_generated()
}

func TestConstraintHeaderDoesntMatch(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names().and().
		a_content_type_constraint_is_added("text/plain")

	when.
		a_request_is_sent_using_the_name("sam")

	then.
		pact_verification_is_not_successful()
}

func TestWaitForPact(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names()

	when.
		multiple_requests_are_sent(50)

	then.
		pact_verification_is_successful().and().
		the_proxy_waits_for_all_requests()
}

func TestModifiedStatusCode(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names().and().
		a_modified_response_status_of_(http.StatusInternalServerError)

	when.
		a_request_is_sent_using_the_name("sam")

	then.
		pact_verification_is_successful().and().
		the_response_is_(http.StatusInternalServerError)
}

func TestModifiedStatusCodeOnARequestWithoutBody(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names_and_returns_no_body().and().
		a_modified_response_status_of_(http.StatusInternalServerError)

	when.
		a_request_is_sent_using_the_name("sam")

	then.
		pact_verification_is_successful().and().
		the_response_is_(http.StatusInternalServerError)
}

func TestModifiedBody(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names().and().
		a_modified_response_body_of_("$.body.name", "jane")

	when.
		a_request_is_sent_using_the_name("sam")

	then.
		pact_verification_is_successful().and().
		the_response_name_is_("jane")
}

func TestModifiedStatusCode_ForNRequests(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names().and().
		a_modified_response_status_of_(http.StatusInternalServerError).and().
		a_modified_response_attempt_of(2)

	when.
		n_requests_are_sent_using_the_name(3, "sam")

	then.
		pact_verification_is_successful().and().
		n_responses_were_received(3).and().
		the_nth_response_is_(1, http.StatusOK).and().
		the_nth_response_is_(2, http.StatusInternalServerError).and().
		the_nth_response_is_(3, http.StatusOK)
}

func TestModifiedBody_ForNRequests(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names().and().
		a_modified_response_body_of_("$.body.name", "jim").and().
		a_modified_response_attempt_of(2)

	when.
		n_requests_are_sent_using_the_name(3, "sam")

	then.
		pact_verification_is_successful().and().
		n_responses_were_received(3).and().
		the_nth_response_name_is_(1, "any").and().
		the_nth_response_name_is_(2, "jim").and().
		the_nth_response_name_is_(3, "any")
}

func TestModifiedBody_With_Numeric_attributes_ForNRequests(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_age().and().
		a_modified_response_body_of_("$.body.age", 14).and().
		a_modified_response_attempt_of(2)

	when.
		n_requests_are_sent_using_the_age(3, 100)

	then.
		pact_verification_is_successful().and().
		n_responses_were_received(3).and().
		the_nth_response_age_is_(1, 100).and().
		the_nth_response_age_is_(2, 14).and().
		the_nth_response_age_is_(3, 100)
}

func TestModifiedBodyWithFirstAndLastName_ForNRequests(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_first_and_last_names().and().
		a_modified_response_body_of_("$.body.first_name", "jim").and().
		a_modified_response_body_of_("$.body.last_name", "gud").and().
		a_modified_response_attempt_of(2)

	when.
		n_requests_are_sent_using_the_body(3, `{"first_name":"sam","last_name":"brown"}`)

	then.
		pact_verification_is_successful().and().
		n_responses_were_received(3).and().
		the_nth_response_body_has_(1, "first_name", "any").and().
		the_nth_response_body_has_(1, "last_name", "any").and().
		the_nth_response_body_has_(2, "first_name", "jim").and().
		the_nth_response_body_has_(2, "last_name", "gud").and().
		the_nth_response_body_has_(3, "first_name", "any").and().
		the_nth_response_body_has_(3, "last_name", "any")
}

type nonJsonTestCase struct {
	reqContentType  string
	reqBody         string
	respContentType string
	respBody        string
}

func createNonJsonTestCases() map[string]nonJsonTestCase {
	return map[string]nonJsonTestCase{
		// text/plain
		"text/plain request and text/plain response": {
			reqContentType:  "text/plain",
			reqBody:         "req text",
			respContentType: "text/plain",
			respBody:        "resp text",
		},
		"text/plain request and application/json response": {
			reqContentType:  "text/plain",
			reqBody:         "req text",
			respContentType: "application/json",
			respBody:        `{"status":"ok"}`,
		},
		// csv
		"text/csv request and text/csv response": {
			reqContentType:  "text/csv",
			reqBody:         "firstname,lastname\nfoo,bar",
			respContentType: "text/csv",
			respBody:        "status,name\n200,ok",
		},
		"text/csv request and application/json response": {
			reqContentType:  "text/csv",
			reqBody:         "firstname,lastname\nfoo,bar",
			respContentType: "application/json",
			respBody:        `{"status":"ok"}`,
		},
		// xml
		"application/xml request and text/csv response": {
			reqContentType:  "application/xml",
			reqBody:         "<root><firstname>foo</firstname></root>",
			respContentType: "application/xml",
			respBody:        "<root><status>200</status></root>",
		},
		"application/xml request and application/json response": {
			reqContentType:  "application/xml",
			reqBody:         "<root><firstname>foo</firstname></root>",
			respContentType: "application/json",
			respBody:        `{"status":"ok"}`,
		},
	}
}

func TestNonJsonContentType(t *testing.T) {
	for testName, tc := range createNonJsonTestCases() {
		t.Run(testName, func(t *testing.T) {
			given, when, then := NewProxyStage(t)

			given.
				a_pact_that_expects(tc.reqContentType, tc.reqBody, tc.respContentType, tc.respBody)

			when.
				a_request_is_sent_with(tc.reqContentType, tc.reqBody)

			then.
				pact_verification_is_successful().and().
				the_response_is_(http.StatusOK).and().
				the_response_body_is(tc.respBody)
		})
	}

}

func TestNonJsonWithModifiedStatusCode(t *testing.T) {
	for testName, tc := range createNonJsonTestCases() {
		t.Run(testName, func(t *testing.T) {
			given, when, then := NewProxyStage(t)

			given.
				a_pact_that_expects(tc.reqContentType, tc.reqBody, tc.respContentType, tc.respBody).and().
				a_modified_response_status_of_(http.StatusInternalServerError)

			when.
				a_request_is_sent_with(tc.reqContentType, tc.reqBody)

			then.
				pact_verification_is_successful().and().
				the_response_is_(http.StatusInternalServerError).and().
				the_response_body_is(tc.respBody)
		})
	}
}

func TestNonJsonWithModifiedResponseBody(t *testing.T) {
	for testName, tc := range createNonJsonTestCases() {
		t.Run(testName, func(t *testing.T) {
			given, when, then := NewProxyStage(t)

			given.
				a_pact_that_expects(tc.reqContentType, tc.reqBody, tc.respContentType, tc.respBody).and().
				a_modified_response_body_of_("$.bytes.body", []byte("newbody"))

			when.
				a_request_is_sent_with(tc.reqContentType, tc.reqBody)

			then.
				pact_verification_is_successful().and().
				the_response_body_is("newbody")
		})
	}
}

func TestNonJsonConstraintMatches(t *testing.T) {
	for testName, tc := range createNonJsonTestCases() {
		t.Run(testName, func(t *testing.T) {
			given, when, then := NewProxyStage(t)

			given.
				a_pact_that_expects(tc.reqContentType, tc.reqBody, tc.respContentType, tc.respBody)

			when.
				a_body_constraint_is_added(tc.reqBody).and().
				a_request_is_sent_with(tc.reqContentType, tc.reqBody)

			then.
				pact_verification_is_successful().and().
				the_response_is_(http.StatusOK).and().
				the_response_body_is(tc.respBody)
		})
	}
}

func TestNonJsonDefaultConstraintAdded(t *testing.T) {
	for testName, tc := range createNonJsonTestCases() {
		t.Run(testName, func(t *testing.T) {
			given, when, then := NewProxyStage(t)

			given.
				a_pact_that_expects(tc.reqContentType, tc.reqBody, tc.respContentType, tc.respBody)

			when.
				a_request_is_sent_with("text/plain", "request with doesn't match constraint")

			then.
				pact_verification_is_not_successful().and().
				the_response_is_(http.StatusBadRequest)
		})
	}
}

func TestNonJsonConstraintDoesNotMatch(t *testing.T) {
	for testName, tc := range createNonJsonTestCases() {
		t.Run(testName, func(t *testing.T) {
			given, when, then := NewProxyStage(t)

			given.
				a_pact_that_expects(tc.reqContentType, tc.reqBody, tc.respContentType, tc.respBody)

			when.
				a_body_constraint_is_added("incorrect file content").and().
				a_request_is_sent_with(tc.reqContentType, tc.reqBody)

			then.
				pact_verification_is_not_successful().and().
				the_response_is_(http.StatusBadRequest)
		})
	}
}

func TestIncorrectContentTypes(t *testing.T) {
	for contentType, wantResponse := range map[string]int{
		"image/bmp":      http.StatusUnsupportedMediaType,
		"invalid format": http.StatusBadRequest,
	} {
		t.Run(contentType, func(t *testing.T) {
			given, when, then := NewProxyStage(t)

			given.
				a_pact_that_expects_plain_text()

			when.
				n_requests_are_sent_using_the_body_and_content_type(1, "req body", contentType)

			then.
				pact_verification_is_not_successful().and().
				the_response_is_(wantResponse).and()
		})
	}
}

func TestEmptyContentTypeDefaultsToPlainText(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_expects_plain_text_without_request_content_type_header()

	when.
		n_requests_are_sent_using_the_body_and_content_type(1, "text", "")

	then.
		pact_verification_is_successful().and().
		the_response_is_(http.StatusOK).and().
		the_response_body_to_plain_text_request_is_correct()
}

func TestGetInteractionDetailsAndHistory(t *testing.T) {
	config := ProxyConfig{RecordHistory: true}
	given, when, then := NewProxyStageWithConfig(t, config)

	given.
		the_record_history_config_option_is_enabled().and().
		a_pact_that_allows_any_names()

	when.
		multiple_requests_are_sent_using_the_names("rod", "jane", "freddy")

	then.
		pact_verification_is_successful().and().
		the_proxy_waits_for_all_requests().and().
		the_proxy_returns_details_of_all_requests()
}

type arrayRequestBodyTestCase struct {
	reqBody         []interface{}
	respContentType string
	respBody        string

	matchedReqBody   string
	unmatchedReqBody string

	matchedConstraintPath    string
	matchedConstraintValue   string
	unmatchedConstraintPath  string
	unmatchedConstraintValue string
}

func createArrayRequestBodyTestCases() map[string]arrayRequestBodyTestCase {
	return map[string]arrayRequestBodyTestCase{
		"array of integers": {
			reqBody:         []interface{}{0, 1, 2},
			respContentType: "application/json",
			respBody:        `[1]`,

			matchedReqBody:   `[0, 1, 2]`,
			unmatchedReqBody: `[1, 2]`,

			matchedConstraintPath:    "$.body[0]",
			matchedConstraintValue:   "0",
			unmatchedConstraintPath:  "$.body[1]",
			unmatchedConstraintValue: "2",
		},
		"array of strings": {
			reqBody:         []interface{}{"a", "b", "c"},
			respContentType: "application/json",
			respBody:        `["ok"]`,

			matchedReqBody:   `["a", "b", "c"]`,
			unmatchedReqBody: `["a", "b", "c", "d"]`,

			matchedConstraintPath:    "$.body[0]",
			matchedConstraintValue:   "a",
			unmatchedConstraintPath:  "$.body[1]",
			unmatchedConstraintValue: "c",
		},
		"array of bools": {
			reqBody:         []interface{}{true, false, true},
			respContentType: "application/json",
			respBody:        `[true]`,

			matchedReqBody:   `[true, false, true]`,
			unmatchedReqBody: `[true, true, true]`,

			matchedConstraintPath:    "$.body[0]",
			matchedConstraintValue:   "true",
			unmatchedConstraintPath:  "$.body[1]",
			unmatchedConstraintValue: "true",
		},
		"array of objects": {
			reqBody: []interface{}{
				map[string]string{"key": "val"},
				map[string]string{"key": "val"},
			},
			respContentType: "application/json",
			respBody:        `[{"status":"ok"}]`,

			matchedReqBody:   `[ {"key": "val"}, {"key": "val"} ]`,
			unmatchedReqBody: `[ {"key": "val"}, {"key": "unexpected value"} ]`,

			matchedConstraintPath:    "$.body[0].key",
			matchedConstraintValue:   "val",
			unmatchedConstraintPath:  "$.body[1].key",
			unmatchedConstraintValue: "wrong value",
		},
	}
}

func TestArrayBodyRequest(t *testing.T) {
	for testName, tc := range createArrayRequestBodyTestCases() {
		t.Run(testName, func(t *testing.T) {
			given, when, then := NewProxyStage(t)

			given.
				a_pact_that_expects("application/json", tc.reqBody, tc.respContentType, tc.respBody)

			when.
				a_request_is_sent_with("application/json", tc.matchedReqBody)

			then.
				pact_verification_is_successful().and().
				the_response_is_(http.StatusOK).and().
				the_response_body_is(tc.respBody)
		})
	}
}

func TestArrayBodyRequestWithModifiedStatusCode(t *testing.T) {
	for testName, tc := range createArrayRequestBodyTestCases() {
		t.Run(testName, func(t *testing.T) {
			given, when, then := NewProxyStage(t)

			given.
				a_pact_that_expects("application/json", tc.reqBody, tc.respContentType, tc.respBody).and().
				a_modified_response_status_of_(http.StatusInternalServerError)

			when.
				a_request_is_sent_with("application/json", tc.matchedReqBody)

			then.
				pact_verification_is_successful().and().
				the_response_is_(http.StatusInternalServerError).and().
				the_response_body_is(tc.respBody)
		})
	}
}

func TestArrayBodyRequestUnmatchedRequestBody(t *testing.T) {
	for testName, tc := range createArrayRequestBodyTestCases() {
		t.Run(testName, func(t *testing.T) {
			given, when, then := NewProxyStage(t)

			given.
				a_pact_that_expects("application/json", tc.reqBody, tc.respContentType, tc.respBody)

			when.
				a_request_is_sent_with("application/json", tc.unmatchedReqBody)

			then.
				// Pact Proxy returns 400 if request body does not match
				pact_verification_is_not_successful().and().
				the_response_is_(http.StatusBadRequest)
		})
	}
}

func TestArrayBodyRequestConstraintMatches(t *testing.T) {
	for testName, tc := range createArrayRequestBodyTestCases() {
		t.Run(testName, func(t *testing.T) {
			given, when, then := NewProxyStage(t)

			given.
				a_pact_that_expects("application/json", tc.reqBody, tc.respContentType, tc.respBody)

			when.
				an_additional_constraint_is_added(tc.matchedConstraintPath, tc.matchedConstraintValue).and().
				a_request_is_sent_with("application/json", tc.matchedReqBody)

			then.
				pact_verification_is_successful().and().
				the_response_is_(http.StatusOK).and().
				the_response_body_is(tc.respBody)
		})
	}
}

func TestArrayBodyRequestConstraintDoesNotMatch(t *testing.T) {
	for testName, tc := range createArrayRequestBodyTestCases() {
		t.Run(testName, func(t *testing.T) {
			given, when, then := NewProxyStage(t)

			given.
				a_pact_that_expects("application/json", tc.reqBody, tc.respContentType, tc.respBody)

			when.
				an_additional_constraint_is_added(tc.unmatchedConstraintPath, tc.unmatchedConstraintValue).and().
				a_request_is_sent_with("application/json", tc.matchedReqBody)

			then.
				pact_verification_is_not_successful().and().
				the_response_is_(http.StatusBadRequest)
		})
	}
}

func TestArrayNestedWithinBody(t *testing.T) {
	pactReqBody := map[string]interface{}{
		"entries": []interface{}{
			map[string]string{"key": "val"},
			map[string]string{"key": "val"},
		},
	}
	const respContentType = "application/json"
	const respBody = `[{"status":"ok"}]`

	const matchedReqBody = `{"entries": [ {"key": "val"}, {"key": "val"} ]}`
	const unmatchedReqBody = `{"entries": [ {"key": "val"}, {"key": "unexpected value"} ]}`
	const unmatchedReqBodyAdditionalArrElem = `{"entries": [ {"key": "val"}, {"key": "val"}, {"key": "val"} ]}`

	const matchedConstraintPath = "$.body.entries[0].key"
	const matchedConstraintValue = "val"

	const unmatchedConstraintPath = "$.body.entries[1].key"
	const unmatchedConstraintValue = "wrong value"

	const outOfBoundsConstraintPath = "$.body.entries[2].key"

	t.Run("Matches", func(t *testing.T) {
		given, when, then := NewProxyStage(t)

		given.
			a_pact_that_expects("application/json", pactReqBody, respContentType, respBody)

		when.
			a_request_is_sent_with("application/json", matchedReqBody)

		then.
			pact_verification_is_successful().and().
			the_response_is_(http.StatusOK).and().
			the_response_body_is(respBody)
	})

	t.Run("Does not match", func(t *testing.T) {
		given, when, then := NewProxyStage(t)

		given.
			a_pact_that_expects("application/json", pactReqBody, respContentType, respBody)

		when.
			a_request_is_sent_with("application/json", unmatchedReqBody)

		then.
			pact_verification_is_not_successful().and().
			the_response_is_(http.StatusBadRequest)
	})

	t.Run("Does not match - additional array element", func(t *testing.T) {
		given, when, then := NewProxyStage(t)

		given.
			a_pact_that_expects("application/json", pactReqBody, respContentType, respBody)

		when.
			a_request_is_sent_with("application/json", unmatchedReqBodyAdditionalArrElem)

		then.
			pact_verification_is_not_successful().and().
			the_response_is_(http.StatusBadRequest)
	})

	t.Run("Matches with additional constraint", func(t *testing.T) {
		given, when, then := NewProxyStage(t)

		given.
			a_pact_that_expects("application/json", pactReqBody, respContentType, respBody).and().
			an_additional_constraint_is_added(matchedConstraintPath, matchedConstraintValue)

		when.
			a_request_is_sent_with("application/json", matchedReqBody)

		then.
			pact_verification_is_successful().and().
			the_response_is_(http.StatusOK).and().
			the_response_body_is(respBody)
	})

	t.Run("Does not match due to additional constraint with different value", func(t *testing.T) {
		given, when, then := NewProxyStage(t)

		given.
			a_pact_that_expects("application/json", pactReqBody, respContentType, respBody).and().
			an_additional_constraint_is_added(unmatchedConstraintPath, unmatchedConstraintValue)

		when.
			a_request_is_sent_with("application/json", matchedReqBody)

		then.
			pact_verification_is_not_successful().and().
			the_response_is_(http.StatusBadRequest)
	})

	t.Run("Does not match due to additional constraint out of array bounds", func(t *testing.T) {
		given, when, then := NewProxyStage(t)

		given.
			a_pact_that_expects("application/json", pactReqBody, respContentType, respBody).and().
			an_additional_constraint_is_added(outOfBoundsConstraintPath, matchedConstraintValue)

		when.
			a_request_is_sent_with("application/json", matchedReqBody)

		then.
			pact_verification_is_not_successful().and().
			the_response_is_(http.StatusBadRequest)
	})
}

// Matching rules are not enforced by Pact proxy, so this test is checking that constraints aren't being
// applied when matching rules are specified, leading to the request being handled by Pact server.
// Note: these tests are using pact DSL style matching rules. Pact proxy identifies these, but differently to
// ones specified in the "matchingRules" JSON property. Loading of the "matchingRules" JSON property style
// gets unit tested in TestLoadInteractionJSONConstraints.
func TestArrayNestedWithinBodyContainingMatchers_NoConstraint(t *testing.T) {
	pactReqBody := map[string]interface{}{
		"entries": []interface{}{
			map[string]any{"key": "val"},
			map[string]any{"key": dsl.Term("a", "(a|b)")},
		},
	}
	const respContentType = "application/json"
	const respBody = `[{"status":"ok"}]`

	const matchedReqBody = `{"entries": [ {"key": "val"}, {"key": "a"} ]}`
	const unmatchedReqBody = `{"entries": [ {"key": "val"}, {"key": "c"} ]}`

	t.Run("Matches", func(t *testing.T) {
		given, when, then := NewProxyStage(t)

		given.
			a_pact_that_expects("application/json", pactReqBody, respContentType, respBody)

		when.
			a_request_is_sent_with("application/json", matchedReqBody)

		then.
			pact_verification_is_successful().and().
			the_response_is_(http.StatusOK).and().
			the_response_body_is(respBody)
	})

	t.Run("Does not match", func(t *testing.T) {
		given, when, then := NewProxyStage(t)

		given.
			a_pact_that_expects("application/json", pactReqBody, respContentType, respBody)

		when.
			a_request_is_sent_with("application/json", unmatchedReqBody)

		then.
			pact_verification_is_not_successful().and().
			// Since the request is passed to Pact Server, the response for not matching is 500
			the_response_is_(http.StatusInternalServerError)
	})
}

func TestEmptyArrayNestedWithinBody(t *testing.T) {
	pactReqBody := map[string]interface{}{
		"entries": []interface{}{},
	}
	const respContentType = "application/json"
	const respBody = `[{"status":"ok"}]`

	const matchedReqBody = `{"entries": []}`
	const unmatchedReqBody = `{"entries": [ {"key": "val"} ]}`

	t.Run("Matches", func(t *testing.T) {
		given, when, then := NewProxyStage(t)

		given.
			a_pact_that_expects("application/json", pactReqBody, respContentType, respBody)

		when.
			a_request_is_sent_with("application/json", matchedReqBody)

		then.
			pact_verification_is_successful().and().
			the_response_is_(http.StatusOK).and().
			the_response_body_is(respBody)
	})

	t.Run("Does not match", func(t *testing.T) {
		given, when, then := NewProxyStage(t)

		given.
			a_pact_that_expects("application/json", pactReqBody, respContentType, respBody)

		when.
			a_request_is_sent_with("application/json", unmatchedReqBody)

		then.
			pact_verification_is_not_successful().and().
			the_response_is_(http.StatusBadRequest)
	})
}
