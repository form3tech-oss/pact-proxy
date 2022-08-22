package app

import (
	"net/http"
	"testing"
)

func TestLargePactResponse(t *testing.T) {
	given, when, then, teardown := NewProxyStage(t)
	defer teardown()

	given.a_pact_for_large_string_generation()

	when.
		a_request_is_sent_to_generate_large_string()

	then.
		pact_verification_is_successful().and().
		the_nth_response_body_has_(1, "generated", largeString)
}

func TestLargePactResponseWithModifiedBody(t *testing.T) {
	given, when, then, teardown := NewProxyStage(t)
	defer teardown()

	given.a_pact_for_large_string_generation().and().
		a_modified_response_body_of_("$.body.name", "jane")

	when.
		n_requests_are_sent_with_modifiers_using_the_body(postLargeStringPact, 1, `{"string":"large"}`)

	then.
		pact_verification_is_successful().and().
		the_response_name_is_("jane").and()
}

func TestLargePactResponseWithModifiedStatusCode(t *testing.T) {
	given, when, then, teardown := NewProxyStage(t)
	defer teardown()

	given.a_pact_for_large_string_generation().and().
		a_modified_response_status_of_(http.StatusInternalServerError)

	when.n_requests_are_sent_with_modifiers_using_the_body(postLargeStringPact, 1, `{"string":"large"}`)

	then.the_response_is_(http.StatusInternalServerError).and().
		pact_verification_is_successful()
}

func TestConstraintMatches(t *testing.T) {
	given, when, then, teardown := NewProxyStage(t)
	defer teardown()

	given.a_pact_that_allows_any_names()

	when.
		a_constaint_is_added("sam").and().
		a_request_is_sent_using_the_name("sam")

	then.
		pact_verification_is_successful().and().
		pact_can_be_generated()
}

func TestConstraintDoesntMatch(t *testing.T) {
	given, when, then, teardown := NewProxyStage(t)
	defer teardown()

	given.a_pact_that_allows_any_names()

	when.
		a_constaint_is_added("sam").and().
		a_request_is_sent_using_the_name("bob")

	then.pact_verification_is_not_successful()
}

func TestWaitForPact(t *testing.T) {
	given, when, then, teardown := NewProxyStage(t)
	defer teardown()

	given.a_pact_that_allows_any_names()

	when.multiple_requests_are_sent(50)

	then.the_proxy_waits_for_all_requests()
}

func TestWaitForAllPacts(t *testing.T) {
	given, when, then, teardown := NewProxyStage(t)
	defer teardown()

	given.
		a_pact_that_allows_any_names().and().
		a_pact_that_allows_any_address()

	when.requests_for_names_and_addresse_are_sent()

	then.the_proxy_waits_for_all_requests()
}

func TestModifiedStatusCode(t *testing.T) {
	given, when, then, teardown := NewProxyStage(t)
	defer teardown()

	given.
		a_pact_that_allows_any_names().and().
		a_modified_response_status_of_(http.StatusInternalServerError)

	when.a_request_is_sent_with_modifiers_using_the_name(postNamePact, "sam")

	then.the_response_is_(http.StatusInternalServerError).and().
		pact_verification_is_successful()
}

func TestModifiedStatusCodeOnARequestWithoutBody(t *testing.T) {
	given, when, then, teardown := NewProxyStage(t)
	defer teardown()

	given.
		a_pact_that_returns_no_body().and().
		a_modified_response_status_of_(http.StatusInternalServerError)

	when.a_request_is_sent_with_modifiers_using_the_name(postNamePact, "sam")

	then.the_response_is_(http.StatusInternalServerError).and().
		pact_verification_is_successful()
}

func TestModifiedBody(t *testing.T) {
	given, when, then, teardown := NewProxyStage(t)
	defer teardown()

	given.
		a_pact_that_allows_any_names().and().
		a_modified_response_body_of_("$.body.name", "jane")

	when.a_request_is_sent_with_modifiers_using_the_name(postNamePact, "sam")

	then.the_response_name_is_("jane").and().
		pact_verification_is_successful()
}

func TestModifiedStatusCode_ForNRequests(t *testing.T) {
	given, when, then, teardown := NewProxyStage(t)
	defer teardown()

	given.
		a_pact_that_allows_any_names().and().
		a_modified_response_status_of_(http.StatusInternalServerError).and().
		a_modified_response_attempt_of(2)

	when.n_requests_are_sent_with_modifiers_using_the_name(postNamePact, 3, "sam")

	then.
		n_responses_were_received(3).and().
		the_nth_response_is_(1, http.StatusOK).and().
		the_nth_response_is_(2, http.StatusInternalServerError).and().
		the_nth_response_is_(3, http.StatusOK).and().
		pact_verification_is_successful()
}

func TestModifiedBody_ForNRequests(t *testing.T) {
	given, when, then, teardown := NewProxyStage(t)
	defer teardown()

	given.
		a_pact_that_allows_any_names().and().
		a_modified_response_body_of_("$.body.name", "jim").and().
		a_modified_response_attempt_of(2)

	when.n_requests_are_sent_with_modifiers_using_the_name(postNamePact, 3, "sam")

	then.
		n_responses_were_received(3).and().
		the_nth_response_name_is_(1, "any").and().
		the_nth_response_name_is_(2, "jim").and().
		the_nth_response_name_is_(3, "any").and().
		pact_verification_is_successful()
}

func TestModifiedBodyWithFirstAndLastName_ForNRequests(t *testing.T) {
	given, when, then, teardown := NewProxyStage(t)
	defer teardown()

	given.
		a_pact_that_allows_any_first_and_last_names().and().
		a_modified_response_body_of_("$.body.first_name", "jim").and().
		a_modified_response_body_of_("$.body.last_name", "gud").and().
		a_modified_response_attempt_of(2)

	when.n_requests_are_sent_with_modifiers_using_the_body(postNamePact, 3, `{"first_name":"sam","last_name":"brown"}`)

	then.
		n_responses_were_received(3).and().
		the_nth_response_body_has_(1, "first_name", "any").and().
		the_nth_response_body_has_(1, "last_name", "any").and().
		the_nth_response_body_has_(2, "first_name", "jim").and().
		the_nth_response_body_has_(2, "last_name", "gud").and().
		the_nth_response_body_has_(3, "first_name", "any").and().
		the_nth_response_body_has_(3, "last_name", "any").and().
		pact_verification_is_successful()
}
