package main

import (
	"testing"
	"time"

	"github.com/theolujay/snippetbox/internal/assert"
)

func TestHumanDate(t *testing.T) {
	tests := []struct {
		name string
		tm   time.Time
		want string
	}{
		{
			name: "UTC",
			tm:   time.Date(2026, 5, 5, 6, 2, 0, 0, time.UTC),
			want: "Tue, 05 May 2026 06:02:00 UTC",
		},
		{
			name: "Empty",
			tm:   time.Time{},
			want: "",
		},
		{
			name: "GET",
			tm:   time.Date(2026, 5, 5, 6, 2, 0, 0, time.FixedZone("WAT", 1*60*60)),
			want: "Tue, 05 May 2026 05:02:00 UTC",
		},
	}

	for _, tt := range tests {

		// t.Runn() runs a sub-test for each test case; its first param is the name
		// of the test (used to identify the sub-test in any log output) and second
		// param is an anonymous function containing the actual test for each case.
		t.Run(tt.name, func(t *testing.T) {
			hd := humanDate(tt.tm)
			assert.Equal(t, hd, tt.want)
		})
	}
}
