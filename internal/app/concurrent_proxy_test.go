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
