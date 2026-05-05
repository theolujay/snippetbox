package main

import (
	"net/http"
	"testing"

	"github.com/theolujay/snippetbox/internal/assert"
)

func TestPing(t *testing.T) {

	app := newTestApplication(t)

	// Create and start a new test server that listens on a randomly-chosen
	// port of the local machine for the duration of the test.
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	// ts.URL contains the network address the test server is listening on.
	statusCode, _, body := ts.get(t, "/ping")

	assert.Equal(t, statusCode, http.StatusOK)
	assert.Equal(t, string(body), "OK")
}
