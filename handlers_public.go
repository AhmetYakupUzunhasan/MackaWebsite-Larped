package main

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

func (a *App) handleHome(w http.ResponseWriter, r *http.Request, lang Language) {
	settings, err := getSiteSettings(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load site settings")
		return
	}
	page, err := getPage(r.Context(), a.db, "home", lang)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load home page")
		return
	}
	sections, err := getSectionsForPage(r.Context(), a.db, "home", lang)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load sections")
		return
	}
	posts, err := listPostCards(r.Context(), a.db, false, 3)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load posts")
		return
	}
	data := HomePageData{
		Settings:     settings,
		Lang:         lang,
		Page:         page,
		Sections:     sections,
		LatestPosts:  posts,
		FlashSuccess: r.URL.Query().Get("success"),
	}
	a.render(w, "templates/home.html", data)
}

func (a *App) handlePage(w http.ResponseWriter, r *http.Request, slug string, lang Language) {
	settings, err := getSiteSettings(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load site settings")
		return
	}
	page, err := getPage(r.Context(), a.db, slug, lang)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		a.renderError(w, http.StatusInternalServerError, "failed to load page")
		return
	}
	a.render(w, "templates/page.html", PageData{Settings: settings, Lang: lang, Page: page, FlashSuccess: r.URL.Query().Get("success")})
}

func (a *App) handleAnnouncements(w http.ResponseWriter, r *http.Request, lang Language) {
	settings, err := getSiteSettings(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load site settings")
		return
	}
	posts, err := listPostCards(r.Context(), a.db, false, 0)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load announcements")
		return
	}
	a.render(w, "templates/posts.html", PostListData{
		Settings: settings,
		Lang:     lang,
		Page:     Page{Title: settings.NavPostsTR},
		Posts:    posts,
	})
}

func (a *App) handleAnnouncementDetail(w http.ResponseWriter, r *http.Request, lang Language, slug string) {
	settings, err := getSiteSettings(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load site settings")
		return
	}
	post, err := getPostBySlug(r.Context(), a.db, slug, false)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		a.renderError(w, http.StatusInternalServerError, "failed to load announcement")
		return
	}
	var media *MediaAsset
	if post.CoverImageID.Valid {
		media, err = getMedia(r.Context(), a.db, post.CoverImageID.Int64)
		if err != nil {
			a.renderError(w, http.StatusInternalServerError, "failed to load media")
			return
		}
	}
	a.render(w, "templates/post_detail.html", PostDetailData{
		Settings: settings,
		Lang:     lang,
		Page: Page{
			Title:          post.TitleTR,
			SEODescription: post.SummaryTR,
		},
		Post:  post,
		Media: media,
	})
}

func (a *App) handleContactSubmit(w http.ResponseWriter, r *http.Request, lang Language) {
	if err := r.ParseForm(); err != nil {
		a.renderError(w, http.StatusBadRequest, "invalid form")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	subject := strings.TrimSpace(r.FormValue("subject"))
	message := strings.TrimSpace(r.FormValue("message"))
	if name == "" || email == "" || message == "" {
		target := pagePath(lang, "contact") + "?success=" + url.QueryEscape("Lütfen zorunlu alanları doldurun.")
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	err := saveContactSubmission(r.Context(), a.db, ContactSubmission{
		Name:     name,
		Email:    email,
		Subject:  subject,
		Message:  message,
		Language: lang,
	})
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to save contact form")
		return
	}
	target := pagePath(lang, "contact") + "?success=" + url.QueryEscape("Mesajınız alındı.")
	http.Redirect(w, r, target, http.StatusSeeOther)
}
