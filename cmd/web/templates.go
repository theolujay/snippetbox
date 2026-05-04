package main

import (
	"html/template"
	"path/filepath"
	"time"

	"github.com/theolujay/snippetbox/internal/models"
)

// templateData acts as the holding structure for any dynamic
// data that we want to pass to our HTML templates.
type templateData struct {
	CurrentYear     int
	Snippet         *models.Snippet
	Snippets        []*models.Snippet
	Form            any
	Flash           string
	IsAuthenticated bool
	CSRFToken       string
}

func humanDate(t time.Time) string {
	return t.Format("02 Jan 2026 at 15:04")
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

	pages, err := filepath.Glob("./ui/html/pages/*.tmpl")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		// The template.FuncMap must be registered with the template set before
		// ParseFiles() method is called. Use template.New() to create an empty
		// template set, use the Funcs() method to register template.FuncMap,
		// and then parse the file as normal.
		ts, err := template.New(name).Funcs(functions).ParseFiles("./ui/html/base.tmpl")
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseGlob("./ui/html/partials/*.tmpl")
		if err != nil {
			return nil, err
		}

		ts, err = ts.ParseFiles(page)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
