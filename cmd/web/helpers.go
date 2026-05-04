package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/go-playground/form/v4"
)

type neuteredFileSystem struct {
	fs http.FileSystem
}

// Implements a surgical approach to disable http.FileServer Directory Listings.
// It returns the index.html if no file in specific is targeted - a directory -
// or returns Not Found when file target doesn't exist in the directory.
func (nfs neuteredFileSystem) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if s.IsDir() {
		// return the (likely empty) index.html file instead
		index := filepath.Join(path, "index.html")
		if _, err := nfs.fs.Open(index); err != nil {
			// avoid a file descriptor leak
			closeErr := f.Close()
			if closeErr != nil {
				return nil, closeErr
			}
			return nil, err
		}
	}
	return f, nil
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

// serverError helper writes an error message and stack trace to the errorLog,
// then sends a generic 500 Internal Server Error response to the user.
func (app *application) serverError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	// Output helps
	app.errorLog.Output(2, trace)

	http.Error(w, http.StatusText(
		http.StatusInternalServerError), http.StatusInternalServerError,
	)
}

// clientError helper sends a specific status code and corresponding description
// to the user.
func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

// render retrieves the appropriate template set from the cache based on the page
// name (like 'home.tmpl') If no entry exists in the cache with the provided name,
// then create a new error and call serverError()
func (app *application) render(
	w http.ResponseWriter, status int, page string, data *templateData,
) {
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s doesn't exist", page)
		app.serverError(w, err)
		return
	}

	buf := new(bytes.Buffer)

	// Write the template to the buffer, instead of straight to the
	// http.ResponseWriter. If there's an error, call serverError()
	err := ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.serverError(w, err)
		return
	}
	// If the template is written to the buffer without any errors, it's save
	// to go ahead and write the HTTP status code to http.ResponseWriter
	w.WriteHeader(status)
	// Write the contents of the buffer to the http.ResponseWriter
	buf.WriteTo(w)
}

func (app *application) newTemplateData(r *http.Request) *templateData {
	return &templateData{
		CurrentYear: time.Now().Year(),
		// PopString() retrives the value for the "flash" key and then deletes
		// the key and value from the session data. It's like a one-time fetch
		// in this use case. It returns an empty string if key doesn't exist.
		Flash: app.sessionManager.PopString(r.Context(), "flash"),
	}
}

func (app *application) decodePostForm(r *http.Request, dst any) error {
	err := r.ParseForm()
	if err != nil {
		err = app.formDecoder.Decode(dst, r.PostForm)
		// If we try to use an invalid target destination, the Decode() will return
		// an error with the type *form.InvalidDecoderError. Use errors.As() to
		// check for this and raise a panic rather than returning the error
		var invalidDecoderError *form.InvalidDecoderError

		if errors.As(err, &invalidDecoderError) {
			// recoverPanic() middleware handles this on the way back up
			panic(err)
		}
		// For all other errors, return them as normal
		return err
	}

	return nil
}
