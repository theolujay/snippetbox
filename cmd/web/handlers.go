package main

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
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

	/*
		Initialize a new createSnippetForm instance and pass it to the template.
		It's also a good opportunity to set any default or 'initial' values for
		the form
	*/
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

	// Call the Decode() method, passing in the current request and a pointer to the
	// snippetCreateForm struct. This will essentially fill the struct with the
	// relevant values from the HTML form.
	if err := app.formDecoder.Decode(&form, r.PostForm); err != nil {
		app.clientError(w, http.StatusBadRequest)
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
		validator.PermittedInt(form.Expires, 1, 7, 365),
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

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	path := filepath.Clean(r.URL.Path)

	http.ServeFile(w, r, path)
}
