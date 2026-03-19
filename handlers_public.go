package main

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

func (a *App) handleHome(w http.ResponseWriter, r *http.Request, lang Language) {
	settings, err := a.cachedSettings(r.Context())
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Site ayarlar\u0131 y\u00fcklenemedi.")
		return
	}
	page, err := a.cachedPage(r.Context(), "home", lang)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Ana sayfa y\u00fcklenemedi.")
		return
	}
	sections, err := a.cachedSections(r.Context(), "home", lang)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "B\u00f6l\u00fcmler y\u00fcklenemedi.")
		return
	}
	posts, err := a.cachedPublishedPosts(r.Context(), 3)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Duyurular y\u00fcklenemedi.")
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
	settings, err := a.cachedSettings(r.Context())
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Site ayarlar\u0131 y\u00fcklenemedi.")
		return
	}
	page, err := a.cachedPage(r.Context(), slug, lang)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		a.renderError(w, http.StatusInternalServerError, "Sayfa y\u00fcklenemedi.")
		return
	}
	a.render(w, "templates/page.html", PageData{Settings: settings, Lang: lang, Page: page, FlashSuccess: r.URL.Query().Get("success")})
}

func (a *App) handleAnnouncements(w http.ResponseWriter, r *http.Request, lang Language) {
	settings, err := a.cachedSettings(r.Context())
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Site ayarlar\u0131 y\u00fcklenemedi.")
		return
	}
	posts, err := a.cachedPublishedPosts(r.Context(), 0)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Duyurular y\u00fcklenemedi.")
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
	settings, err := a.cachedSettings(r.Context())
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Site ayarlar\u0131 y\u00fcklenemedi.")
		return
	}
	post, media, err := a.cachedPostDetail(r.Context(), slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		a.renderError(w, http.StatusInternalServerError, "Duyuru y\u00fcklenemedi.")
		return
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
		a.renderError(w, http.StatusBadRequest, "Form verisi okunamad\u0131.")
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	email := strings.TrimSpace(r.FormValue("email"))
	subject := strings.TrimSpace(r.FormValue("subject"))
	message := strings.TrimSpace(r.FormValue("message"))
	if name == "" || email == "" || message == "" {
		target := pagePath(lang, "contact") + "?success=" + url.QueryEscape("L\u00fctfen zorunlu alanlar\u0131 doldurun.")
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
		a.renderError(w, http.StatusInternalServerError, "\u0130leti\u015fim formu kaydedilemedi.")
		return
	}
	target := pagePath(lang, "contact") + "?success=" + url.QueryEscape("Mesaj\u0131n\u0131z al\u0131nd\u0131.")
	http.Redirect(w, r, target, http.StatusSeeOther)
}
