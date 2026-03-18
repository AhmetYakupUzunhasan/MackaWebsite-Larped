package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type App struct {
	config    Config
	db        *sql.DB
	templates *template.Template
}

type contextKey string

const adminUserKey contextKey = "adminUser"

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(a.config.UploadDir))))
	mux.HandleFunc("/admin/login", a.handleAdminLogin)
	mux.HandleFunc("/admin/logout", a.handleAdminLogout)
	mux.Handle("/admin/", a.requireAdmin(http.HandlerFunc(a.adminRouter)))
	mux.HandleFunc("/", a.publicRouter)
	return mux
}

func (a *App) publicRouter(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/admin/") {
		http.NotFound(w, r)
		return
	}

	if r.URL.Path == "/en" || r.URL.Path == "/en/" {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/en/") {
		http.Redirect(w, r, strings.TrimPrefix(r.URL.Path, "/en"), http.StatusMovedPermanently)
		return
	}

	lang, slugPath := languageFromPath(r.URL.Path)
	switch {
	case r.Method == http.MethodGet && slugPath == "":
		a.handleHome(w, r, lang)
	case slugPath == "contact" && r.Method == http.MethodPost:
		a.handleContactSubmit(w, r, lang)
	case slugPath == "about" && r.Method == http.MethodGet:
		a.handlePage(w, r, "about", lang)
	case slugPath == "contact" && r.Method == http.MethodGet:
		a.handlePage(w, r, "contact", lang)
	case slugPath == "announcements" && r.Method == http.MethodGet:
		a.handleAnnouncements(w, r, lang)
	case strings.HasPrefix(slugPath, "announcements/") && r.Method == http.MethodGet:
		a.handleAnnouncementDetail(w, r, lang, path.Base(slugPath))
	default:
		http.NotFound(w, r)
	}
}

func (a *App) adminRouter(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/admin/" || r.URL.Path == "/admin":
		a.handleAdminDashboard(w, r)
	case r.URL.Path == "/admin/settings":
		a.handleAdminSettings(w, r)
	case r.URL.Path == "/admin/pages":
		a.handleAdminPages(w, r)
	case r.URL.Path == "/admin/pages/edit":
		a.handleAdminPageEdit(w, r)
	case r.URL.Path == "/admin/sections":
		a.handleAdminSections(w, r)
	case r.URL.Path == "/admin/sections/edit":
		a.handleAdminSectionEdit(w, r)
	case r.URL.Path == "/admin/posts":
		a.handleAdminPosts(w, r)
	case r.URL.Path == "/admin/posts/new":
		a.handleAdminPostNew(w, r)
	case r.URL.Path == "/admin/posts/edit":
		a.handleAdminPostEdit(w, r)
	case r.URL.Path == "/admin/posts/publish" && r.Method == http.MethodPost:
		a.handleAdminPostPublish(w, r)
	case r.URL.Path == "/admin/posts/delete" && r.Method == http.MethodPost:
		a.handleAdminPostDelete(w, r)
	case r.URL.Path == "/admin/media":
		a.handleAdminMedia(w, r)
	case r.URL.Path == "/admin/media/delete" && r.Method == http.MethodPost:
		a.handleAdminMediaDelete(w, r)
	case r.URL.Path == "/admin/contacts":
		a.handleAdminContacts(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (a *App) render(w http.ResponseWriter, templateName string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, templateName, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) renderError(w http.ResponseWriter, status int, message string) {
	http.Error(w, message, status)
}

func (a *App) currentAdmin(r *http.Request) *AdminUser {
	user, _ := r.Context().Value(adminUserKey).(*AdminUser)
	return user
}

func (a *App) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(a.config.SessionName)
		if err != nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		id, err := strconv.ParseInt(cookie.Value, 10, 64)
		if err != nil {
			a.clearSession(w)
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		user, err := getAdminUserByID(r.Context(), a.db, id)
		if err != nil || user == nil {
			a.clearSession(w)
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), adminUserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) signIn(w http.ResponseWriter, user *AdminUser) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.config.SessionName,
		Value:    strconv.FormatInt(user.ID, 10),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *App) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.config.SessionName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *App) setFlash(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *App) readFlash(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("flash")
	if err != nil {
		return ""
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return cookie.Value
}

func languageFromPath(urlPath string) (Language, string) {
	cleaned := strings.Trim(strings.TrimSpace(urlPath), "/")
	if cleaned == "" {
		return LangTR, ""
	}
	return LangTR, cleaned
}

func parseID(raw string) int64 {
	id, _ := strconv.ParseInt(raw, 10, 64)
	return id
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func saveUploadedFile(file io.Reader, target string) error {
	dst, err := os.Create(target)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, file)
	return err
}

func mediaURL(fileName string) string {
	if strings.TrimSpace(fileName) == "" {
		return ""
	}
	return "/uploads/" + fileName
}

func adminLayoutData(ctx context.Context, db *sql.DB, title string, user *AdminUser) (AdminLayoutData, error) {
	settings, err := getSiteSettings(ctx, db)
	if err != nil {
		return AdminLayoutData{}, err
	}
	return AdminLayoutData{Title: title, User: user, Settings: settings}, nil
}

func fileNameForUpload(original string) string {
	ext := path.Ext(original)
	base := strings.TrimSuffix(original, ext)
	return fmt.Sprintf("%s-%d%s", slugify(base), time.Now().UnixNano(), strings.ToLower(ext))
}
