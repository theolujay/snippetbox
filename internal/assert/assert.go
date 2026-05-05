package assert

import (
	"testing"
)

func Equal[T comparable](t *testing.T, actual, expected T) {
	// Mark as helper function, and skip during logs.
	// Go test runner will report in the output the filename
	// and line number of the code which calls this helper.
	t.Helper()

	if actual != expected {
		t.Errorf("got: %v; want: %v", actual, expected)
	}
}
