package app

import (
	"testing"
	"time"
)

func TestConcurrentRequestsForDifferentModifiersHaveTheCorrectResponses(t *testing.T) {
	given, when, then := NewConcurrentProxyStage(t)

	given.
		a_pact_that_allows_any_names().and().
		a_modified_name_status_code().and().
		a_pact_that_allows_any_address().and().
		a_modified_address_status_code()

	when.
		x_concurrent_user_requests_per_second_are_made_for_y_seconds(5, 5*time.Second).and().
		x_concurrent_address_requests_per_second_are_made_for_y_seconds(5, 5*time.Second).and().
		the_concurrent_requests_are_sent()

	then.
		all_the_user_responses_should_have_the_right_status_code().and().
		all_the_address_responses_should_have_the_right_status_code()
}

func TestConcurrentRequestsForSameModifierBasedOnAttempt(t *testing.T) {
	given, when, then := NewConcurrentProxyStage(t)
	given.
		a_pact_that_allows_any_names()
	when.
		the_concurrent_requests_are_sent_with_attempt_based_modifier()
	then.
		the_second_user_response_should_have_the_right_status_code_and_body()
}

func TestMultipleModifiersForSameInteraction(t *testing.T) {
	given, when, then := NewConcurrentProxyStage(t)
	given.
		a_pact_that_allows_any_names()
	when.
		the_concurrent_requests_are_sent_with_multiple_modifiers_for_same_interaction()
	then.
		all_responses_should_have_the_expected_return_values()

}

func TestConcurrentRequestsWaitForAllPacts(t *testing.T) {
	given, when, then := NewConcurrentProxyStage(t)

	given.
		a_pact_that_allows_any_names().and().
		a_pact_that_allows_any_address()

	when.
		x_concurrent_user_requests_per_second_are_made_for_y_seconds(5, 5*time.Second).and().
		x_concurrent_address_requests_per_second_are_made_for_y_seconds(5, 5*time.Second).and().
		the_concurrent_requests_are_sent()

	then.
		the_proxy_waits_for_all_user_responses().and().
		the_proxy_waits_for_all_address_responses()
}
