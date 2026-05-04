package main

import (
	"crypto/tls"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/theolujay/snippetbox/internal/models"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/go-sql-driver/mysql"
)

type application struct {
	errorLog       *log.Logger
	infoLog        *log.Logger
	snippets       *models.SnippetModel
	users          *models.UserModel
	templateCache  map[string]*template.Template
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
}

func main() {
	addr := flag.String("addr", ":4000", "HTTP network address")

	dsn := flag.String("dsn", "web:pass@/snippetbox?parseTime=true", "MySQL data source name")

	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime|log.LUTC)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.LUTC|log.Llongfile)

	db, err := openDB(*dsn)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer db.Close()

	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}

	formDecoder := form.NewDecoder()

	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour
	// Setting this means that the cokie will only be sent by a user's
	// web browser when a HTTPS connection is n use (and won't be sent
	// over unsecure HTTP connection)
	sessionManager.Cookie.Secure = true

	app := &application{
		errorLog:       errorLog,
		infoLog:        infoLog,
		snippets:       &models.SnippetModel{DB: db},
		users:          &models.UserModel{DB: db},
		templateCache:  templateCache,
		formDecoder:    formDecoder,
		sessionManager: sessionManager,
	}

	// Initialize a tls.Config struct to hold the non-default TLS settings
	// the server should use. The only thing changes is the curve preference
	// value, so that only elliptic curves with assembly implementatons are
	// used
	tlsConfig := &tls.Config{
		// Restrict to only the curves with assembly implementations.
		// This keeps handshake cost low under heavy load.
		CurvePreferences: []tls.CurveID{
			tls.X25519,    // Bernstein's curve, fast and clean
			tls.CurveP256, // NIST P-256, universally supported, assembly-backed
		},
		// Min and max TLS versions can be configured, especially in situations
		// where one knows that certain computers to be used support a specific
		// version -- say TLS 1.2
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12,
		// For some applications, it may be desirable to limit the HTTPS
		// server to only support some of these cipher suites. For example, it
		// might be desirable to only support cipher suites which use ECDHE
		// (forward secrecy) and not support weak cipher suites that use RC4,
		// 3DES or CBC. One can do this via the tls.Config.CipherSuites field.
		// Go will automatically choose which is actually used based on the
		// client/server hardware support.
		// It’s also important (and interesting) to note that if a TLS 1.3
		// connection is negotiated, any CipherSuites field in your tls.Config
		// will be ignored. The reason for this is that all the cipher suites that
		// Go supports for TLS 1.3 connections are considered to be safe, so
		// there isn’t much point in providing a mechanism to configure them.
		// Basically, using tls.Config to set a custom list of supported cipher
		// suites will affect TLS 1.0-1.2 connections only.
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	srv := &http.Server{
		Addr:         *addr,
		ErrorLog:     errorLog,
		Handler:      app.routes(),
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		// ReadHeaderTimeout could also be set. In cases of Slowloris attacks,
		// it might be useful to consider in conjunction with ReadTimeout.
	}

	infoLog.Printf("Starting server on %s", *addr)
	// ListenAndServeTLS() starts the HTTPS server, with the paths to the
	// TLS certificate and corresponding private key as the two parameters
	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	errorLog.Fatal(err)
}
