package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/justinas/nosurf"
)

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(
			"Content-Security-Policy",
			"default-src 'self'; style-src 'self' fonts.googleapis.com; font-src fonts.gstatic.com",
		)
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")

		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.infoLog.Printf(
			"%s - %s %s %s", r.RemoteAddr, r.Proto, r.Method, r.URL.RequestURI(),
		)

		next.ServeHTTP(w, r)
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				/*
					Setting the Connection: Close header on the response acts as a
					trigger to make Go’s HTTP server automatically close the current
					connection after a response has been sent. It also informs the
					user that the connection will be closed. Note: If the protocol being
					used is HTTP/2, Go will automatically strip the Connection: Close
					header from the response (so it is not malformed) and send a
					GOAWAY frame.
				*/
				w.Header().Set("Connection", "close")
				app.serverError(w, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !app.isAuthenticated(r) {
			app.sessionManager.Put(r.Context(), "redirectPathAfterLogin", r.URL.Path)
			http.Redirect(w, r, "/user/login", http.StatusSeeOther)
			return
		}

		// Set "Cache-Control: no-store" header to that pages that require
		// authentication are not stored in the user's browser cache (or
		// other intermediary cache)
		w.Header().Add("Cache-Control", "no-store")

		next.ServeHTTP(w, r)
	})
}

func noSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)
	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	})

	return csrfHandler
}

type contextKey string

const isAuthenticatedContextKey = contextKey("isAuthenticated")

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Retrieve the authenticatedUserID value from the session using the
		// GetInt() method. This will return the value for an int (0) if no
		// "authentiicatedUserID" value is in the session -- in which case
		// the next handler in the chain is called as normal and return.
		id := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
		if id == 0 {
			next.ServeHTTP(w, r)
			return
		}
		// Otherwise check to see if a user with that ID exists in the database
		exists, err := app.users.Exists(id)
		if err != nil {
			app.serverError(w, err)
			return
		}
		// If a matching user is found, we know that the request is coming from
		// an authenticated use who exists in the database. Create a copy of the
		// request (with an isAuthenticatedContextKey value of true in the request
		// context) and assign it to r.
		if exists {
			ctx := context.WithValue(r.Context(), isAuthenticatedContextKey, true)
			r = r.WithContext(ctx)
		}
		// If we don't have a valid authenticated user, the original and unchanged
		// *http.Request is passed to the next handler in the chain. When we do
		// have a valid authenticated user, the copy of the request with
		// `isAuthenticatedContextKey` key and `true` value is stored in the request
		// context.
		next.ServeHTTP(w, r)
	})
}
