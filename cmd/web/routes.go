package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"

	"github.com/theolujay/snippetbox/ui"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()
	// httprouter allows setting custom response, like for 404 and 405. This
	// is nice for consistency
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})
	router.MethodNotAllowed = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.clientError(w, http.StatusMethodNotAllowed)
	})
	// Convert the ui.Files into https.FS type so it satisfies the http.FileSystem interface.
	fileServer := http.FileServer(http.FS(ui.Files))
	// Since the static files are contained in the "static" folder of the ui.Files embedded
	// filesystem, there's no need to strip the prefix from the request URL -->
	// http.StripPrefix("/static", fileServer)
	router.Handler(http.MethodGet, "/static/*filepath", fileServer)

	dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf, app.authenticate)

	router.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.home))
	router.Handler(http.MethodGet, "/snippet/view/:id", dynamic.ThenFunc(app.snippetView))
	router.Handler(http.MethodGet, "/user/signup", dynamic.ThenFunc(app.userSignup))
	router.Handler(http.MethodPost, "/user/signup", dynamic.ThenFunc(app.userSignupPost))
	router.Handler(http.MethodGet, "/user/login", dynamic.ThenFunc(app.userLogin))
	router.Handler(http.MethodPost, "/user/login", dynamic.ThenFunc(app.userLoginPost))

	// Protected (authenticated-only) application routes
	protected := dynamic.Append(app.requireAuthentication)
	router.Handler(http.MethodGet, "/snippet/create", protected.ThenFunc(app.snippetCreate))
	router.Handler(http.MethodPost, "/snippet/create", protected.ThenFunc(app.snippetCreatePost))
	router.Handler(http.MethodPost, "/user/logout", protected.ThenFunc(app.userLogoutPost))

	standard := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	return standard.Then(router)
}
