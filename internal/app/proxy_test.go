package app

import (
	"net/http"
	"testing"
)

func TestLargePactResponse(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_for_large_string_generation()

	when.
		a_request_is_sent_to_generate_large_string()

	then.
		pact_verification_is_successful().and().
		the_nth_response_body_has_(1, "generated", largeString)
}

func TestLargePactResponseWithModifiedBody(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_for_large_string_generation().and().
		a_modified_response_body_of_("$.body.name", "jane")

	when.
		n_requests_are_sent_with_modifiers_using_the_body(1, `{"string":"large"}`)

	then.
		pact_verification_is_successful().and().
		the_response_name_is_("jane").and()
}

func TestLargePactResponseWithModifiedStatusCode(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_for_large_string_generation().and().
		a_modified_response_status_of_(http.StatusInternalServerError)

	when.
		n_requests_are_sent_with_modifiers_using_the_body(1, `{"string":"large"}`)

	then.
		pact_verification_is_successful().and().
		the_response_is_(http.StatusInternalServerError)
}

func TestConstraintMatches(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names()

	when.
		a_constraint_is_added("sam").and().
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
		a_constraint_is_added("sam").and().
		a_request_is_sent_using_the_name("bob")

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

func TestWaitForAllPacts(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_names().and().
		a_pact_that_allows_any_address()

	when.
		requests_for_names_and_addresse_are_sent()

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
		a_request_is_sent_with_modifiers_using_the_name("sam")

	then.
		pact_verification_is_successful().and().
		the_response_is_(http.StatusInternalServerError)
}

func TestModifiedStatusCodeOnARequestWithoutBody(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_returns_no_body().and().
		a_modified_response_status_of_(http.StatusInternalServerError)

	when.
		a_request_is_sent_with_modifiers_using_the_name("sam")

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
		a_request_is_sent_with_modifiers_using_the_name("sam")

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
		n_requests_are_sent_with_modifiers_using_the_name(3, "sam")

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
		n_requests_are_sent_with_modifiers_using_the_name(3, "sam")

	then.
		pact_verification_is_successful().and().
		n_responses_were_received(3).and().
		the_nth_response_name_is_(1, "any").and().
		the_nth_response_name_is_(2, "jim").and().
		the_nth_response_name_is_(3, "any")
}

func TestModifiedBodyWithFirstAndLastName_ForNRequests(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_allows_any_first_and_last_names().and().
		a_modified_response_body_of_("$.body.first_name", "jim").and().
		a_modified_response_body_of_("$.body.last_name", "gud").and().
		a_modified_response_attempt_of(2)

	when.
		n_requests_are_sent_with_modifiers_using_the_body(3, `{"first_name":"sam","last_name":"brown"}`)

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

func TestTextPlainContentType(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_expects_plain_text()

	when.
		a_request_is_sent_in_plain_text()

	then.
		pact_verification_is_successful().and().
		the_response_is_(http.StatusOK).and().
		the_response_body_to_plain_text_request_is_correct()
}

func TestModifiedStatusCodeWithPlainTextBody(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_expects_plain_text().and().
		a_modified_response_status_of_(http.StatusInternalServerError)

	when.
		a_request_is_sent_in_plain_text()

	then.
		pact_verification_is_successful().and().
		the_response_is_(http.StatusInternalServerError).and().
		the_response_body_to_plain_text_request_is_correct()
}

func TestPlainTextConstraintMatches(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_expects_plain_text()

	when.
		a_constraint_is_added("text").and().
		a_request_is_sent_in_plain_text()

	then.
		pact_verification_is_successful().and().
		the_response_is_(http.StatusOK).and().
		the_response_body_to_plain_text_request_is_correct()
}

func TestPlainTextDefaultConstraintAdded(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_expects_plain_text()

	when.
		a_request_is_sent_in_plain_text_with_body("request with doesn't match constraint")

	then.
		pact_verification_is_not_successful().and().
		the_response_is_(http.StatusBadRequest)
}

func TestPlainTextConstraintDoesNotMatch(t *testing.T) {
	given, when, then := NewProxyStage(t)

	given.
		a_pact_that_expects_plain_text()

	when.
		a_constraint_is_added("incorrect file content").and().
		a_request_is_sent_in_plain_text()

	then.
		pact_verification_is_not_successful().and().
		the_response_is_(http.StatusBadRequest)
}

func TestPlainTextDifferentRequestAndResponseBodies(t *testing.T) {
	given, when, then := NewProxyStage(t)

	requestConstraint := "request body"

	given.
		// TODO fix this test to have dynamic params
		a_pact_that_expects_plain_text_with_different_request_response()

	when.
		a_constraint_is_added(requestConstraint).and().
		a_request_is_sent_in_plain_text_with_body("request body")

	then.
		pact_verification_is_successful().and().
		the_response_is_(http.StatusOK).and().
		the_response_body_is([]byte("response body"))
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
				a_request_is_sent_with_body_and_content_type("req body", contentType)

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
		a_request_is_sent_with_body_and_content_type("text", "")

	then.
		pact_verification_is_successful().and().
		the_response_is_(http.StatusOK).and().
		the_response_body_to_plain_text_request_is_correct()
}
