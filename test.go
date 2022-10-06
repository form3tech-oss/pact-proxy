package test

import "testing"

func TestFatal(t *testing.T) {
	go func() {
		t.Fatal("oops") // This exits the inner func instead of TestFoo.
	}()
}

func TestFatal2(t *testing.T) {
	go func() {
		fatal(t) // This should raise a vet warning but does not.
	}()
}

func fatal(t *testing.T) {
	t.Fatal("oops") // This exits the inner func instead of TestFoo.
}

