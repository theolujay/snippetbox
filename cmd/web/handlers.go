package main

import (
	"errors"
	"fmt"
	"net/http"

	// "path/filepath"
	"strconv"

	"github.com/theolujay/snippetbox/internal/models"
	"github.com/theolujay/snippetbox/internal/validator"

	"github.com/julienschmidt/httprouter"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	snippets, err := app.snippets.Latest()
	if err != nil {
		app.serverError(w, err)
		return
	}

	data := app.newTemplateData(r)
	data.Snippets = snippets

	app.render(w, http.StatusOK, "home.tmpl", data)
}

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	app.render(w, http.StatusOK, "about.tmpl", data)
}

func (app *application) snippetView(w http.ResponseWriter, r *http.Request) {
	// httprouter stores values of named parameters in the request context when
	// parsing the request
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.Atoi(params.ByName("id"))

	if err != nil || id < 1 {
		app.notFound(w)
		return
	}
	snippet, err := app.snippets.Get(id)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return
	}

	data := app.newTemplateData(r)
	data.Snippet = snippet

	app.render(w, http.StatusOK, "view.tmpl", data)
}

func (app *application) snippetCreate(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	// Initialize a new createSnippetForm instance and pass it to the template.
	// It's also a good opportunity to set any default or 'initial' values for
	// the form
	data.Form = snippetCreateForm{
		Expires: 365,
	}

	app.render(w, http.StatusOK, "create.tmpl", data)
}

// The struct tags tell the form.Decoder how to map HTML form values into the
// different struct fields. For example, the decoder is told to store the value
// from the HTML form input with the name "title" in the Title field. `form:"-"`
// tells the decoder to completely ignore a field during decoding.
type snippetCreateForm struct {
	Title               string `form:"title"`
	Content             string `form:"content"`
	Expires             int    `form:"expires"`
	validator.Validator `form:"-"`
}

func (app *application) snippetCreatePost(w http.ResponseWriter, r *http.Request) {
	var form snippetCreateForm
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.Checkfield(
		validator.NotBlank(form.Title),
		"title", "This field cannot be blank",
	)
	form.Checkfield(
		validator.MaxChars(form.Title, 100),
		"title",
		"This field cannot be more than 100 characters",
	)
	form.Checkfield(
		validator.NotBlank(form.Content),
		"content",
		"This field cannot be blank",
	)
	form.Checkfield(
		validator.PermittedValue(form.Expires, 1, 7, 365),
		"expires",
		"This field must equal 1, 7 or 365",
	)

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "create.tmpl", data)
		return
	}

	id, err := app.snippets.Insert(form.Title, form.Content, form.Expires)
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Snippet successfully created!")

	http.Redirect(w, r, fmt.Sprintf("/snippet/view/%d", id), http.StatusSeeOther)
}

// func downloadHandler(w http.ResponseWriter, r *http.Request) {
// 	path := filepath.Clean(r.URL.Path)

// 	http.ServeFile(w, r, path)
// }

type userSignupForm struct {
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userSignupForm{}
	app.render(w, http.StatusOK, "signup.tmpl", data)
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	var form userSignupForm
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.Checkfield(
		validator.NotBlank(form.Name),
		"name", "This field cannot be blank",
	)
	form.Checkfield(
		validator.NotBlank(form.Email),
		"email",
		"This field cannot be blank",
	)
	form.Checkfield(
		validator.Matches(form.Email, validator.EmailRX),
		"email",
		"This field must be a valid email address",
	)
	form.Checkfield(
		validator.NotBlank(form.Password),
		"password",
		"This field cannot be blank",
	)
	form.Checkfield(
		validator.MinChars(form.Password, 8),
		"password",
		"This field must be at least 8 characters",
	)

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "signup.tmpl", data)
		return
	}

	err = app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address already in use")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "signup.tmpl", data)
		} else {
			app.serverError(w, err)
		}
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "Signup successful! Please log in.")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

type userLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, http.StatusOK, "login.tmpl", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	var form userLoginForm
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.Checkfield(
		validator.NotBlank(form.Email),
		"email",
		"This field cannot be blank",
	)
	form.Checkfield(
		validator.Matches(form.Email, validator.EmailRX),
		"email",
		"This field must be a valid email address",
	)
	form.Checkfield(
		validator.NotBlank(form.Password),
		"password",
		"This field cannot be blank",
	)
	form.Checkfield(
		validator.MinChars(form.Password, 8),
		"password",
		"This field must be at least 8 characters",
	)

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, http.StatusUnprocessableEntity, "login.tmpl", data)
		return
	}

	id, err := app.users.Authenticate(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, http.StatusUnprocessableEntity, "login.tmpl", data)
		} else {
			app.serverError(w, err)
		}
		return
	}

	// Change the sesion ID, as it's good practice to generate a new session ID
	// when the authentication state or privilege level changes for the user
	// (e.g. login and logout operations)
	// Note: The SessionManager.RenewToken() method that we’re
	// using in the code above will change the ID of the current user’s
	// session but retain any data associated with the session. It’s
	// good practice to do this before login to mitigate the risk of a
	// session fixation attack. For more background and information on this
	// please see the OWASP Session Management Cheat Sheet
	// (https://github.com/OWASP/CheatSheetSeries/blob/master/cheatsheets/Session_Management_Cheat_Sheet.md#renew-the-session-id-after-any-privilege-level-change).
	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.sessionManager.Put(r.Context(), "flash", "Login successful!")
	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)

	http.Redirect(w, r, "/snippet/create", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {

	// Use the RenewToken() method on the current session to change the session
	// ID again.
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.sessionManager.Remove(r.Context(), "authenticatedUserID")
	app.sessionManager.Put(r.Context(), "flash", "You've been logged out successfully!")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}
