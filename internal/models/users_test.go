package models

import (
	"testing"

	"github.com/theolujay/snippetbox/internal/assert"
)

func TestUserModelExists(t *testing.T) {
	// Skip this test if the "-short" flag is provided when running tests
	if testing.Short() {
		t.Skip("models: skipping integration test")
	}
	tests := []struct {
		name   string
		userID int
		want   bool
	}{
		{
			name:   "Valid ID",
			userID: 1,
			want:   true,
		},
		{
			name:   "Zero ID",
			userID: 0,
			want:   false,
		},
		{
			name:   "Non-existent ID",
			userID: 2,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calling newTestDB() here -- inside t.Run() -- means that
			// fresh database tables and data will be set up and torn down
			db := newTestDB(t)

			// New instance of UserModel
			m := UserModel{db}

			exists, err := m.Exists(tt.userID)
			assert.Equal(t, exists, tt.want)
			assert.NilError(t, err)
		})
	}
}
