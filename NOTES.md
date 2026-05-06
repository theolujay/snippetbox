# Learning Golang from the book Let's Go

> *These were originally hand-written while I went through the book. They may or may not make complete sense without the book's context.*

- Go respects the version in `go.mod` and would fail to build if not compatible. Read more at [go.dev/ref/mod](https://go.dev/ref/mod).
- Web app basics:
    - Handler: more like controllers, handling HTTP response headers and bodies and executing app logic.
    - Router (servemux): maps URL patters to handlers. There's usually one `servemux` containing all routes.
    - Web server: Go can run without dependencies like Nginx or Apache by establishing a web server within the app itself.
- There's such a thing as "named ports", and they can be found in `/etc/services`. For example, `http` is a named port for port 80, likewise `ssh` for port 22 and `https` for 443. Ports 0-1023 are restricted and only usable by root-privileged services.
- Avoid using `DefaultServeMux`; while Go defaults to this if instantiated, it can be a vunerability to attacks because an external module could hijack it.
- MIME Header implies "MIME-style headers", and it means header names are case-insensitive.
- While `http.FileServer()` handles cleaning up fo paths, some cases may require one to use `filepath.Clean()` when using `http.ServeFile` instead. This is to prevent traversal attacks.

    > A path traversal attack (also known as directory traversal) aims to access files and directories that are stored outside the web root folder. By manipulating variables that reference files with "dot-dot-slash (../)" sequences and its variations or by using absolute file paths, it may be possible to access arbitrary files and directories stored on file system including application source code or configuration and critical system files.<sup>[src](https://owasp.org/www-community/attacks/Path_Traversal)</sup>

- `find ./ui/static -type d` searches for all directories (`-type d`) starting from `./ui/static` and recursively into sub-directories.
- Use a custom `http.FileServer` to disable 'directory listing'.

    <details>
    <summary>Implementation (from <a href="./cmd/web/helpers.go">cmd/web/helpers.go</a>)</summary>

    ```go
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
    ```

    </details>

---

- What's a handler, though? It's an object which satisfies the `http.Handler` interface:
    ```go
    type Handler interface {
        ServeHTTP(ResponseWriter, *Request)
    }
    ```
    - But it's long-winded to use such an approach of creating objects, so rather use use HandlerFunc adapter.
        ```go
        mux := http.NewSeveMux()
        // home implements ServeHTTP so it can be adapted
        mux.Handle("/", http.HandlerFunc(home))
        ```
    - `mux.HandleFunc` is shorter -- although syntatic sugar.
- Requests are handled concurrently, so be aware of [race conditions](https://www.alexedwards.net/blog/understanding-mutexes). Using `sync.Mutex` helps design for for this.
- How are flags joined using bitwise OR operator `|`?
    ```go
    // flags (e.g. log.Ldate) are joined using the bitwise OR operator '|'
    infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
    ```
    The bitwise OR operator `|` combines multiple flag constants into a single integer value. EAch flag is a distinct patter bit position, so *ORing* them together creates a bitmask that represents all the selected options. For example:
    ```go
    log.Ldate = 1 // binary: 0001
    log.Ltime = 8 // binary: 1000
    ```
    When they're OR-ed together:
    ```go
    log.Ldate | log.Ltime
    // 0001 | 1000 = 1001 (binary) = 9 (decimal)
    ```
    The result is a single integer with multiple bit sets, one for each flag selected.

    <details>
    <summary>Why this pattern?</summary>

    Bitwise flags are used because they're:
    - Memory efficient: Store multiple options in a single integer
    - Fast: Just bit operations, no allocations
    - Composable: Easily combine flags with `|`
    - Testable: Check flags with `&`

    It's a common pattern in systems programming and low-level Go code. Also see in file permissions `(os.O_RDONLY | os.O_WRONLY)`, syscall flags, and many stdlib packages.

    </details>

- Avoid using `Panic()` and `Fatal()` outside of the `main()` function. They crash the program immediately, so return errors instead.

    <details>
    <summary>Example</summary>

    Problem:
    ```go
    func processUser(id int) {
        user, err := fetchUser(id)
        if err != nil {
            log.Fatal(err)  // Crashes the entire program
        }
    }
    ```

    Better:
    ```go
    func processUser(id int) error {
        user, err := fetchUser(id)
        if err != nil {
            return fmt.Errorf("failed to fetch user %d: %w", id, err)
        }
        return nil
    }
    ```

    </details>

- Using `log.Output` with call depth of 2 helps to see the line where the error originated from -- right before the logger's call site.

    <details>
    <summary>Implementation (from <a href="./cmd/web/helpers.go">cmd/web/helpers.go</a>)</summary>

    ```go
    // serverError helper writes an error message and stack trace to the errorLog,
    // then sends a generic 500 Internal Server Error response to the user.
    func (app *application) serverError(w http.ResponseWriter, err error) {
        trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
        // Use call depth of 2 to find actual point of failure
        app.errorLog.Output(2, trace)

        if app.debug {
            http.Error(w, trace, http.StatusInternalServerError)
            return
        }
        http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
    }
    ```

    </details>

🏁 CHECKPOINT: I applied some concepts learned so far in [appa](https://github.com/theolujay/appa)

---

- `go.mod` file contains cryptographic checksums representing the content of the required packages. Use `go mod verify` to verify the packages. Use `go mod tidy -v` to remove unused packages, or `go get <module>@none`.
- `sql.Open()` function doesn't create any connections; it only initializes the pool for future use. Use `sql.Open().Ping()` to create a connection and check for any errors.
- `defer db.Close()` (in [cmd/web/main.go](./cmd/web/main.go)) is never actually ran since the program exits at *signal interrupt* or by `errorLog.Fatal()`, but it is good habit to write.
    ```go
    func main()
        ...
        db, err := openDB(*dsn)
        if err != nil {
            errorLog.Fatal(err)
        }
        defer db.Close()
        ...
    ```
- Go database methods:
    - `Query()`; for `SELECT` queries that return <u>multiple</u> rows</u>
    - `QueryRow()`; for `SELECT` queries that return <u>a single row</u>
    - `Exec()`; for queries that don't return rows, like `INSERT AND DELETE`;
        `Exec()` works in three steps:
        1. Creates a new prepared statement, parses and compiles it, then stores it for execution.
        2. Passes the parameter values to the database and then executes the statement using these parameters. Because the parameters are transmitted later, after the statement has been compiled, the database treats them as data. (This prevents injection).
        3. It then closes (or deallocates) the prepared statement on the database.

        Example:
        ```go
        result, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)",
            "Alice", "alice@example.com")
        ```

    NOTE: Always remember to `defer Close()` on database connections, else it'll all be used up.

- `%+v` in something like `fmt.FPrintf(w, "%+v\n", s)` prints struct field names alongside values

    <details>
    <summary>Example</summary>

    ```go
    type User struct {
        Name  string
        Email string
    }

    u := User{"Alice", "alice@example.com"}
    fmt.Printf("%+v\n", u)
    // Output: {Name:Alice Email:alice@example.com}
    ```

    Compare to `%v` (no field names): `{Alice alice@example.com}`

    </details>

- Always call `Commit()` or `Rollback()` before transaction function returns
  - Use `defer tx.Rollback()` for automatic rollback on error

    <details>
    <summary>Example</summary>

    ```go
    tx, err := db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    _, err = tx.Exec("INSERT INTO users (name) VALUES ($1)", "Alice")
    if err != nil {
        return err
    }

    return tx.Commit().Error  // Commits if no errors
    ```

    </details>

---

### Middleware

- Positioning the middleware:
    - Before servemux:
        ```plaintext
        myMiddleware ----> servemux ----> application handler
        ```
        This applies to all requests, like it's used in logging
    - After servemux:
        ```plaintext
        servemux ----> myMiddleware ----> application handler
        ```
        Applies to certain to number of routes, like auth-related handlers
- `Content-Security-Policy` (CSP) is a feature that helps prevent or minimize the risk of certain types of security threats. A few things it helps with are:
    - defending against cross-site scripting (XSS) and clickhijacking.
    - ensuring resources are served over HTTPS.
    - flow of control (after implementing a secureHeaders middleware)
        ```plaintext
        secureHeaders ----> servemux ----> application handler ----> servemux ----> secureHeaders
        ```
- Early returns: If return is called in a middleware function before `next.ServeHTTP()` is called, then the chain will stop being executed and control will flow back upstream.
- Remember to check logs early in web browser dev tools to debug CSP-related issues.
- Logging requests:
    ```plaintext
    logRequest <----> secureHeaders <---->  app handler <----> servemux
    ```
- On recovering panic, `Connection: close` acts as a trigger to make Go's HTTP server automatically close the current connection after a response has been sent. If it's an HTTP/2 connection, Go strips the the `Connection: close` header from the response (so it is not malformed) and rather sends a `GOAWAY` frame.
    <details>
    <summary>What's a GOAWAY frame?</summary>
        The HTTP/2 GOAWAY frame is used by a erver to gracefully stop accepting new streams while allowing existing ones to complete.
    </details>
- Embeddings in GO:
    - Structs in structs
    - Interfaces in interfaces
    - Inferface in structs
    Read me more <a href="https://eli.thegreenplace.net/2020/embedding-in-go-part-1-structs-in-structs/">here</a>
- Configuring HTTPS setting:
    - `Connection: keep-alive` is used to allow clients use the same TCP connection instead of a new one with every request/response cycle. This reduces the number of "hops" and generally improves performance.
    - A `keep-alive` connection is automatically closed after a couple of minutes (with the exact time depending on the OS). And there is no way o increase this -- although it can be reduced -- without rolling out a new `net.Listener`.
- File embeddings allow for one to bundle templates -- UI assets -- into the binary output, removing the need for a filesystem.

    <details>
    <summary>Implementation (from <a href="./ui/efs.go">ui/efs.go</a>)</summary>

    ```go
    import "embed"

    //go:embed "html" "static"
    var Files embed.FS
    ```

    </details>

    <details>
    <summary>Usage in backend (from <a href="./cmd/web/templates.go">cmd/web/templates.go</a> and <a href="./cmd/web/routes.go">cmd/web/routes.go</a>)</summary>

    For template parsing:
    ```go
    // Parse the template files from the ui.Files embedded filesystem
    ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, patterns...)
    ```

    For serving static files:
    ```go
    // Convert ui.Files to http.FileSystem and serve
    fileServer := http.FileServer(http.FS(ui.Files))
    router.Handler(http.MethodGet, "/static/*filepath", fileServer)
    ```

    </details>

- HTTPS is HTTP over TLS (Transport Layer Security, modern version of SSL)
- TLS encrypts and signs data for privacy and integrity in transit

- Self-signed certificates for development:
    - Not signed by a trusted certificate authority (browser warns first time)
    - Still encrypts correctly, fine for dev/testing
    - For production, use <a href="https://letsencrypt.org/">Let's Encrypt</a>

- Generate self-signed certificate using Go's `crypto/tls` package:
    ```bash
    go run /usr/local/go/src/crypto/tls/generate_cert.go --rsa-bits=2048 --host=localhost
    ```
    This generates two files:
    1. `key.pem` — private key (2048-bit RSA)
    2. `cert.pem` — self-signed TLS certificate containing public key

    Both are PEM-encoded (standard format for TLS implementations)

### Testing

- Useful flags for Go Test:
    - `go test -v`: verbose
    - `go test ./...`: run al tests
    - `go test -run 'string'`: run specific tests with the provided expression
    - `go test -skip`: prevent specific tests from running
    - `go test -count=1`: used to tell go test how many times each test should run. It's noncacheable, and a neat trick to run without cache. OR you could clear out the test cache with `go clean -testcache`.
    - `go test -failfast`: for fast failure. It used to stop ONLY the tests in the package that had the failure in `go1.12`, but has since been fixed in `go1.23`.

- Parallel testing:
    ```go
    func TestPing(t *testing.T) {
        t.Parallel() // this indicates that this test can be ran in parallel
    }
    ```
    - Tests marked as parallel will only run in parallel with other parallel tests.
    - Max number of parallel tests is limited by the value of `GOMAXPROCS`, but it can be overriden with `-parallel=4` flag.

- Race detector:
    - Uses the `-race` flag to enable it.
    - Recommnded if concurrency is implemented in the project.
    - Does not run static code analysis; only flags possible racing during testing runtime, and it does guarantee no possible race conditions if none is flagged.
    - It increases overall running time, so it's advised to run it before committing code.
    - Requires CGo or something... I'm not exactly sure what this is about yet.

- Go Tool ignores any directories called 'testdata', as well as any directory or files with names that begin with an `_` or `.` character.
- Skipping long-running tests:
    ```go
    func TestSomething(t *testing.T) {
        // skip this test if the -short flag is used
        if testing.Short() {
            t.Skip("models: skipping integration test")
        }
        ...
    }
    ```
- Profiling test coverage:
    - `go test -cover ./...` -> shows metrics
    - `go test -coverprofile=/tmp/profile.out` -> get more detailed breakdown
    - `go tool cover -func=/tmp/profile.out` -> view the detailed breakdown in the terminal
    - `go tool cover -html=/tmp/profile.out` -> view the detailed breakdown in the browser
    - `go test -covermode=count -coverprofile=/tmp/profile.out ./...` -> record levels of coverage
        - NOTE: If running any test in parallel, use `-covermode=atomic` instead of `-covermode=count`
