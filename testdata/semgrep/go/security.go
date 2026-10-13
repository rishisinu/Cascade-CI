package testdata

import (
	"archive/tar"
	"archive/zip"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func shell(ctx context.Context, userInput string) {
	// ruleid: go-shell-command-injection
	exec.Command("sh", "-c", "git log "+userInput).Run()
	// ruleid: go-shell-command-injection
	exec.CommandContext(ctx, "/bin/bash", "-c", userInput).Run()
	// ok: go-shell-command-injection
	exec.Command("sh", "-c", "echo hello").Run()
	// ok: go-shell-command-injection
	exec.Command("git", "log", userInput).Run()
}

func handler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	// ruleid: go-request-input-to-command
	exec.Command("convert", name).Run()
	// ok: go-request-input-to-command
	exec.Command("convert", "static.png").Run()

	file := r.FormValue("file")
	// ruleid: go-request-input-to-file-path
	os.ReadFile(filepath.Join("/srv/uploads", file))
	// ok: go-request-input-to-file-path
	os.ReadFile(filepath.Join("/srv/uploads", filepath.Base(file)))

	target := r.PathValue("url")
	// ruleid: go-request-input-to-ssrf
	http.Get(target)
	// ok: go-request-input-to-ssrf
	http.Get("https://example.com/health")
}

func queries(ctx context.Context, db *sql.DB, id string) {
	// ruleid: go-sql-string-building
	db.Query(fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", id))
	// ruleid: go-sql-string-building
	db.ExecContext(ctx, "DELETE FROM users WHERE id = "+id)
	q := fmt.Sprintf("SELECT * FROM users WHERE name = '%s'", id)
	// ruleid: go-sql-string-building
	db.QueryRow(q)
	// ok: go-sql-string-building
	db.Query("SELECT * FROM users WHERE id = ?", id)
}

func tlsConfig() {
	// ruleid: go-tls-insecure-skip-verify
	_ = &tls.Config{InsecureSkipVerify: true}
	cfg := &tls.Config{}
	// ruleid: go-tls-insecure-skip-verify
	cfg.InsecureSkipVerify = true
	// ok: go-tls-insecure-skip-verify
	_ = &tls.Config{MinVersion: tls.VersionTLS12}
}

func hashes(b []byte) {
	// ruleid: go-weak-hash
	md5.Sum(b)
	// ok: go-weak-hash
	sha256.Sum256(b)
}

func servers(h http.Handler) {
	// ruleid: go-http-server-without-timeouts
	http.ListenAndServe(":8080", h)
	// ruleid: go-http-server-without-timeouts
	srv := &http.Server{Addr: ":8080", Handler: h}
	// ok: go-http-server-without-timeouts
	safe := &http.Server{Addr: ":8080", Handler: h, ReadHeaderTimeout: 5 * time.Second}
	_, _ = srv, safe
}

func render(userBio string) {
	// ruleid: go-template-html-cast
	_ = template.HTML(userBio)
	// ok: go-template-html-cast
	_ = template.HTML("<b>static</b>")
}

func perms(f *os.File) {
	// ruleid: go-chmod-world-writable
	os.Chmod("/tmp/out", 0777)
	// ruleid: go-chmod-world-writable
	f.Chmod(0o666)
	// ok: go-chmod-world-writable
	os.Chmod("/tmp/out", 0600)
}

func unzip(r *zip.Reader, dest string) {
	for _, f := range r.File {
		// ruleid: go-archive-path-traversal
		path := filepath.Join(dest, f.Name)
		_ = path
	}
	for _, f := range r.File {
		if !filepath.IsLocal(f.Name) {
			continue
		}
		// ok: go-archive-path-traversal
		path := filepath.Join(dest, f.Name)
		_ = path
	}
}

func untar(tr *tar.Reader, dest string) {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		// ruleid: go-archive-path-traversal
		target := filepath.Join(dest, hdr.Name)
		_ = target
	}
}
