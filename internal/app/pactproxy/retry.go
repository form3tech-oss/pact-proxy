package pactproxy

import (
	"errors"
	"time"

	"github.com/avast/retry-go"
)

func retryFor(do func(time.Duration) bool, delay, duration time.Duration) bool {
	start := time.Now()
	err := retry.Do(func() error {
		timeLeft := duration - time.Since(start)
		if !do(timeLeft) {
			return errors.New("retry")
		}
		return nil
	}, retry.Delay(delay), retry.RetryIf(func(err error) bool {
		return err != nil && time.Since(start) <= duration
	}))
	return err != nil
}
