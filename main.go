package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed templates/*.html templates/admin/*.html migrations/*.sql
var embeddedFiles embed.FS

func main() {
	cfg := loadConfig()

	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := runMigrations(context.Background(), db, embeddedFiles); err != nil {
		log.Fatal(err)
	}
	if err := seedData(context.Background(), db, cfg); err != nil {
		log.Fatal(err)
	}

	tpl, err := parseTemplates()
	if err != nil {
		log.Fatal(err)
	}

	app := &App{
		config:    cfg,
		db:        db,
		templates: tpl,
	}

	listener, port, err := listenWithFallback(cfg.Port)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on http://localhost:%s", port)
	log.Fatal(server.Serve(listener))
}

func parseTemplates() (*template.Template, error) {
	funcMap := template.FuncMap{
		"safeHTML": func(v string) template.HTML {
			return template.HTML(v)
		},
		"nl2br":            nl2br,
		"valueByLang":      valueByLang,
		"pagePath":         pagePath,
		"announcementPath": announcementPath,
		"mediaURL":         mediaURL,
		"sectionLabel":     sectionLabel,
		"fmtDate": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("02 Jan 2006")
		},
		"excerpt": func(v string, max int) string {
			v = strings.TrimSpace(v)
			if len(v) <= max {
				return v
			}
			return strings.TrimSpace(v[:max]) + "..."
		},
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("dict expects even arguments")
			}
			m := map[string]any{}
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				m[key] = values[i+1]
			}
			return m, nil
		},
	}

	return template.New("base").Funcs(funcMap).ParseFS(
		embeddedFiles,
		"templates/*.html",
		"templates/admin/*.html",
	)
}

func listenWithFallback(preferredPort string) (net.Listener, string, error) {
	ports := []string{preferredPort}
	if base, err := strconv.Atoi(preferredPort); err == nil {
		for i := 1; i <= 10; i++ {
			ports = append(ports, strconv.Itoa(base+i))
		}
	}

	var lastErr error
	for _, port := range ports {
		ln, err := net.Listen("tcp", ":"+port)
		if err == nil {
			return ln, port, nil
		}
		lastErr = err
	}
	return nil, "", lastErr
}
