package main

import (
	"html/template"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/theolujay/snippetbox/internal/models"
	"github.com/theolujay/snippetbox/ui"
)

// templateData acts as the holding structure for any dynamic
// data that we want to pass to our HTML templates.
type templateData struct {
	CurrentYear     int
	User            *models.User
	Snippet         *models.Snippet
	Snippets        []*models.Snippet
	Form            any
	Flash           string
	IsAuthenticated bool
	CSRFToken       string
}

func humanDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC1123)
}

/*
Custom template functions (like humanDate() function below) can
accept as many parameters as they need to, but they must return
one value only. The only exception to this is if there's a need
to return an error as the second value, in which case that’s OK too.
*/
var functions = template.FuncMap{
	"humanDate": humanDate,
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}
	// Use fs.Glob() to get a slice of all filepaths in the ui.Files embedded
	// filesystem which match the pattern 'html/pages/*.tmpl'. This essentially
	// gives a slice of all the 'page' templates for the application.
	pages, err := fs.Glob(ui.Files, "html/pages/*.tmpl")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		// Create a slice containng the filepath patters to parse
		patterns := []string{
			"html/base.tmpl",
			"html/partials/*.tmpl",
			page,
		}

		// The template.FuncMap must be registered with the template set before
		// ParseFiles() method is called. Use template.New() to create an empty
		// template set, use the Funcs() method to register template.FuncMap,
		// and then parse the file as normal.
		ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
