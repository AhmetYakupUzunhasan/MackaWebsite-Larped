package main

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func (a *App) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if cookie, err := r.Cookie(a.config.SessionName); err == nil && cookie.Value != "" {
			http.Redirect(w, r, "/admin/", http.StatusSeeOther)
			return
		}
		a.render(w, "templates/admin/login.html", map[string]any{
			"Error": r.URL.Query().Get("error"),
		})
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/login?error=Geçersiz+istek", http.StatusSeeOther)
		return
	}
	user, err := getAdminUserByUsername(r.Context(), a.db, r.FormValue("username"))
	if err != nil || user == nil || user.PasswordHash != hashPassword(r.FormValue("password")) {
		http.Redirect(w, r, "/admin/login?error=Kullanıcı+adı+veya+şifre+hatalı", http.StatusSeeOther)
		return
	}
	a.signIn(w, user)
	http.Redirect(w, r, "/admin/", http.StatusSeeOther)
}

func (a *App) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
		return
	}
	a.clearSession(w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *App) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Yönetim Paneli", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load dashboard")
		return
	}
	layout.Flash = a.readFlash(w, r)
	counts, err := getDashboardCounts(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to count records")
		return
	}
	contacts, err := listContactSubmissions(r.Context(), a.db, 5)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load contacts")
		return
	}
	a.render(w, "templates/admin/dashboard.html", DashboardData{
		AdminLayoutData: layout,
		PageCount:       counts["pages"],
		SectionCount:    counts["sections"],
		PostCount:       counts["posts"],
		MediaCount:      counts["media"],
		ContactCount:    counts["contacts"],
		RecentContacts:  contacts,
	})
}

func (a *App) handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Site Ayarları", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load settings")
		return
	}
	settings, err := getSiteSettings(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load settings")
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			a.renderError(w, http.StatusBadRequest, "invalid form")
			return
		}
		settings = SiteSettings{
			ID:                1,
			AssociationNameTR: r.FormValue("association_name_tr"),
			TaglineTR:         r.FormValue("tagline_tr"),
			FooterTextTR:      r.FormValue("footer_text_tr"),
			ContactEmail:      r.FormValue("contact_email"),
			ContactPhone:      r.FormValue("contact_phone"),
			AddressTR:         r.FormValue("address_tr"),
			InstagramURL:      r.FormValue("instagram_url"),
			FacebookURL:       r.FormValue("facebook_url"),
			LinkedInURL:       r.FormValue("linkedin_url"),
			NavHomeTR:         r.FormValue("nav_home_tr"),
			NavAboutTR:        r.FormValue("nav_about_tr"),
			NavContactTR:      r.FormValue("nav_contact_tr"),
			NavPostsTR:        r.FormValue("nav_posts_tr"),
			SEODescriptionTR:  r.FormValue("seo_description_tr"),
		}
		if err := updateSiteSettings(r.Context(), a.db, settings); err != nil {
			a.renderError(w, http.StatusInternalServerError, "failed to save settings")
			return
		}
		a.invalidateContentCache()
		a.setFlash(w, "Site ayarları güncellendi.")
		http.Redirect(w, r, "/admin/settings", http.StatusSeeOther)
		return
	}
	layout.Flash = a.readFlash(w, r)
	a.render(w, "templates/admin/settings.html", AdminSettingsData{AdminLayoutData: layout, Form: settings})
}

func (a *App) handleAdminPages(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Sayfalar", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load pages")
		return
	}
	layout.Flash = a.readFlash(w, r)
	pages, err := listPages(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to query pages")
		return
	}
	a.render(w, "templates/admin/pages.html", AdminPagesData{AdminLayoutData: layout, Pages: pages})
}

func (a *App) handleAdminPageEdit(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Sayfa Düzenle", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load page")
		return
	}
	slug := r.URL.Query().Get("slug")
	if slug == "" {
		slug = "about"
	}
	lang := LangTR
	page, err := getPage(r.Context(), a.db, slug, lang)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		a.renderError(w, http.StatusInternalServerError, "failed to load page")
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		page = Page{Slug: slug, Language: lang}
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			a.renderError(w, http.StatusBadRequest, "invalid form")
			return
		}
		page = Page{
			Slug:           r.FormValue("slug"),
			Language:       Language(r.FormValue("language")),
			Title:          r.FormValue("title"),
			Intro:          r.FormValue("intro"),
			Body:           r.FormValue("body"),
			SEODescription: r.FormValue("seo_description"),
		}
		if err := upsertPage(r.Context(), a.db, page); err != nil {
			a.renderError(w, http.StatusInternalServerError, "failed to save page")
			return
		}
		a.invalidateContentCache()
		a.setFlash(w, "Sayfa güncellendi.")
		http.Redirect(w, r, "/admin/pages", http.StatusSeeOther)
		return
	}
	a.render(w, "templates/admin/page_edit.html", AdminPageEditData{AdminLayoutData: layout, Page: page})
}

func (a *App) handleAdminSections(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Ana Sayfa Bölümleri", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load sections")
		return
	}
	layout.Flash = a.readFlash(w, r)
	sections, err := listSections(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to query sections")
		return
	}
	media, err := listMedia(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to query media")
		return
	}
	a.render(w, "templates/admin/sections.html", AdminSectionsData{AdminLayoutData: layout, Sections: sections, Media: media})
}

func (a *App) handleAdminSectionEdit(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Bölüm Düzenle", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load section")
		return
	}
	id := parseID(r.URL.Query().Get("id"))
	section, err := getSection(r.Context(), a.db, id)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load section")
		return
	}
	media, err := listMedia(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to query media")
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			a.renderError(w, http.StatusBadRequest, "invalid form")
			return
		}
		section.Title = r.FormValue("title")
		section.Subtitle = r.FormValue("subtitle")
		section.Body = r.FormValue("body")
		section.CTAName = r.FormValue("cta_name")
		section.CTAURL = r.FormValue("cta_url")
		section.SortOrder = int(parseID(r.FormValue("sort_order")))
		section.ImageID = nullInt64(parseID(r.FormValue("image_id")))
		if err := updateSection(r.Context(), a.db, section); err != nil {
			a.renderError(w, http.StatusInternalServerError, "failed to save section")
			return
		}
		a.invalidateContentCache()
		a.setFlash(w, "Bölüm güncellendi.")
		http.Redirect(w, r, "/admin/sections", http.StatusSeeOther)
		return
	}
	a.render(w, "templates/admin/section_edit.html", AdminSectionEditData{AdminLayoutData: layout, Section: section, Media: media})
}

func (a *App) handleAdminPosts(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Duyurular", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load posts")
		return
	}
	layout.Flash = a.readFlash(w, r)
	posts, err := listPosts(r.Context(), a.db, true)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to query posts")
		return
	}
	a.render(w, "templates/admin/posts.html", AdminPostsData{AdminLayoutData: layout, Posts: posts})
}

func (a *App) handleAdminPostNew(w http.ResponseWriter, r *http.Request) {
	a.handleAdminPostForm(w, r, 0)
}

func (a *App) handleAdminPostEdit(w http.ResponseWriter, r *http.Request) {
	a.handleAdminPostForm(w, r, parseID(r.URL.Query().Get("id")))
}

func (a *App) handleAdminPostForm(w http.ResponseWriter, r *http.Request, id int64) {
	layout, err := adminLayoutData(r.Context(), a.db, "Duyuru Düzenle", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load post")
		return
	}
	post := Post{}
	if id > 0 {
		post, err = getPostByID(r.Context(), a.db, id)
		if err != nil {
			a.renderError(w, http.StatusInternalServerError, "failed to load post")
			return
		}
	}
	media, err := listMedia(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to query media")
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			a.renderError(w, http.StatusBadRequest, "invalid form")
			return
		}
		coverImageID := nullInt64(parseID(r.FormValue("cover_image_id")))
		file, header, fileErr := r.FormFile("cover_upload")
		if fileErr == nil {
			defer file.Close()
			fileName := fileNameForUpload(header.Filename)
			target := filepath.Join(a.config.UploadDir, fileName)
			if err := saveUploadedFile(file, target); err != nil {
				a.renderError(w, http.StatusInternalServerError, "failed to store file")
				return
			}
			title := r.FormValue("cover_title")
			if title == "" {
				title = header.Filename
			}
			mediaID, err := insertMedia(r.Context(), a.db, MediaAsset{
				Title:        title,
				AltTR:        r.FormValue("cover_alt_tr"),
				AltEN:        "",
				FileName:     fileName,
				OriginalName: header.Filename,
				MimeType:     header.Header.Get("Content-Type"),
			})
			if err != nil {
				a.renderError(w, http.StatusInternalServerError, "failed to save media record")
				return
			}
			a.invalidateContentCache()
			coverImageID = nullInt64(mediaID)
		}
		post = Post{
			ID:           id,
			Slug:         firstNonEmpty(slugify(r.FormValue("slug")), slugify(r.FormValue("title_tr"))),
			TitleTR:      r.FormValue("title_tr"),
			TitleEN:      "",
			SummaryTR:    r.FormValue("summary_tr"),
			SummaryEN:    "",
			BodyTR:       r.FormValue("body_tr"),
			BodyEN:       "",
			CoverImageID: coverImageID,
			Published:    r.FormValue("published") == "on",
		}
		if id > 0 {
			current, err := getPostByID(r.Context(), a.db, id)
			if err == nil {
				post.CreatedAt = current.CreatedAt
				post.PublishedAt = current.PublishedAt
				if !post.Published {
					post.PublishedAt = sqlNullTime{}
				}
			}
		}
		if _, err := savePost(r.Context(), a.db, post); err != nil {
			a.renderError(w, http.StatusInternalServerError, "failed to save post")
			return
		}
		a.invalidateContentCache()
		a.setFlash(w, "Duyuru kaydedildi.")
		http.Redirect(w, r, "/admin/posts", http.StatusSeeOther)
		return
	}
	a.render(w, "templates/admin/post_edit.html", AdminPostEditData{AdminLayoutData: layout, Post: post, Media: media})
}

func (a *App) handleAdminPostDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.renderError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if err := deletePost(r.Context(), a.db, parseID(r.FormValue("id"))); err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to delete post")
		return
	}
	a.invalidateContentCache()
	a.setFlash(w, "Duyuru silindi.")
	http.Redirect(w, r, "/admin/posts", http.StatusSeeOther)
}

func (a *App) handleAdminPostPublish(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.renderError(w, http.StatusBadRequest, "invalid request")
		return
	}
	id := parseID(r.FormValue("id"))
	post, err := getPostByID(r.Context(), a.db, id)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load post")
		return
	}
	post.Published = true
	post.PublishedAt = nullTime(time.Now())
	if _, err := savePost(r.Context(), a.db, post); err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to publish post")
		return
	}
	a.invalidateContentCache()
	a.setFlash(w, "Duyuru yayına alındı.")
	http.Redirect(w, r, "/admin/posts", http.StatusSeeOther)
}

func (a *App) handleAdminMedia(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Medya Kütüphanesi", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load media")
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			a.renderError(w, http.StatusBadRequest, "invalid upload")
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			a.renderError(w, http.StatusBadRequest, "file is required")
			return
		}
		defer file.Close()
		fileName := fileNameForUpload(header.Filename)
		target := filepath.Join(a.config.UploadDir, fileName)
		if err := saveUploadedFile(file, target); err != nil {
			a.renderError(w, http.StatusInternalServerError, "failed to store file")
			return
		}
		_, err = insertMedia(r.Context(), a.db, MediaAsset{
			Title:        r.FormValue("title"),
			AltTR:        r.FormValue("alt_tr"),
			AltEN:        "",
			FileName:     fileName,
			OriginalName: header.Filename,
			MimeType:     header.Header.Get("Content-Type"),
		})
		if err != nil {
			a.renderError(w, http.StatusInternalServerError, "failed to save media record")
			return
		}
		a.invalidateContentCache()
		a.setFlash(w, "Medya yüklendi.")
		http.Redirect(w, r, "/admin/media", http.StatusSeeOther)
		return
	}
	layout.Flash = a.readFlash(w, r)
	media, err := listMedia(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to query media")
		return
	}
	a.render(w, "templates/admin/media.html", AdminMediaData{AdminLayoutData: layout, Media: media})
}

func (a *App) handleAdminMediaDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.renderError(w, http.StatusBadRequest, "invalid request")
		return
	}
	id := parseID(r.FormValue("id"))
	media, err := getMedia(r.Context(), a.db, id)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load media")
		return
	}
	if media != nil {
		_ = deleteMedia(r.Context(), a.db, id)
		_ = os.Remove(filepath.Join(a.config.UploadDir, media.FileName))
	}
	a.invalidateContentCache()
	a.setFlash(w, "Medya silindi.")
	http.Redirect(w, r, "/admin/media", http.StatusSeeOther)
}

func (a *App) handleAdminContacts(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "İletişim Mesajları", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to load contacts")
		return
	}
	contacts, err := listContactSubmissions(r.Context(), a.db, 0)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "failed to query contacts")
		return
	}
	a.render(w, "templates/admin/contacts.html", AdminContactsData{AdminLayoutData: layout, Contacts: contacts})
}
