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
		if user, err := a.authenticatedAdmin(r); err == nil && user != nil {
			http.Redirect(w, r, "/admin/", http.StatusSeeOther)
			return
		}
		a.clearSession(w)
		a.render(w, "templates/admin/login.html", map[string]any{
			"Error": r.URL.Query().Get("error"),
		})
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/login?error=Ge\u00e7ersiz+istek", http.StatusSeeOther)
		return
	}
	user, err := getAdminUserByUsername(r.Context(), a.db, r.FormValue("username"))
	if err != nil || user == nil || user.PasswordHash != hashPassword(r.FormValue("password")) {
		http.Redirect(w, r, "/admin/login?error=Kullan\u0131c\u0131+ad\u0131+veya+\u015fifre+hatal\u0131", http.StatusSeeOther)
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
	layout, err := adminLayoutData(r.Context(), a.db, "Y\u00f6netim Paneli", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Y\u00f6netim paneli y\u00fcklenemedi.")
		return
	}
	layout.Flash = a.readFlash(w, r)
	counts, err := getDashboardCounts(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Kay\u0131t say\u0131lar\u0131 al\u0131namad\u0131.")
		return
	}
	contacts, err := listContactSubmissions(r.Context(), a.db, 5)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Mesajlar y\u00fcklenemedi.")
		return
	}
	a.render(w, "templates/admin/dashboard.html", DashboardData{
		AdminLayoutData: layout,
		PageCount:       counts["pages"],
		SectionCount:    counts["sections"],
		PostCount:       counts["posts"],
		ContactCount:    counts["contacts"],
		RecentContacts:  contacts,
	})
}

func (a *App) handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Site Ayarlar\u0131", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Site ayarlar\u0131 y\u00fcklenemedi.")
		return
	}
	settings, err := getSiteSettings(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Site ayarlar\u0131 y\u00fcklenemedi.")
		return
	}
	passwordError := ""
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			a.renderError(w, http.StatusBadRequest, "Form verisi okunamad\u0131.")
			return
		}
		if r.FormValue("form_name") == "password" {
			admin := a.currentAdmin(r)
			if admin == nil {
				a.renderError(w, http.StatusUnauthorized, "Y\u00f6netici oturumu bulunamad\u0131.")
				return
			}
			currentPassword := r.FormValue("current_password")
			newPassword := r.FormValue("new_password")
			confirmPassword := r.FormValue("confirm_password")
			switch {
			case currentPassword == "":
				passwordError = "L\u00fctfen mevcut \u015fifrenizi girin."
			case hashPassword(currentPassword) != admin.PasswordHash:
				passwordError = "Mevcut \u015fifreniz hatal\u0131."
			case len(newPassword) < 8:
				passwordError = "Yeni \u015fifreniz en az 8 karakter olmal\u0131d\u0131r."
			case newPassword != confirmPassword:
				passwordError = "Yeni \u015fifre ile do\u011frulama birbiriyle e\u015fle\u015fmiyor."
			case currentPassword == newPassword:
				passwordError = "L\u00fctfen mevcut \u015fifreden farkl\u0131 bir \u015fifre se\u00e7in."
			default:
				newHash := hashPassword(newPassword)
				if err := updateAdminUserPassword(r.Context(), a.db, admin.ID, newHash); err != nil {
					a.renderError(w, http.StatusInternalServerError, "\u015eifre g\u00fcncellenemedi.")
					return
				}
				a.clearSession(w)
				a.setFlash(w, "Y\u00f6netici \u015fifresi g\u00fcncellendi. L\u00fctfen yeniden giri\u015f yap\u0131n.")
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
				return
			}
		} else {
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
				a.renderError(w, http.StatusInternalServerError, "Site ayarlar\u0131 kaydedilemedi.")
				return
			}
			a.invalidateContentCache()
			a.setFlash(w, "Site ayarlar\u0131 g\u00fcncellendi.")
			http.Redirect(w, r, "/admin/settings", http.StatusSeeOther)
			return
		}
	}
	layout.Flash = a.readFlash(w, r)
	a.render(w, "templates/admin/settings.html", AdminSettingsData{
		AdminLayoutData: layout,
		Form:            settings,
		PasswordError:   passwordError,
	})
}

func (a *App) handleAdminPassword(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "\u015eifre De\u011fi\u015ftir", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "\u015eifre sayfas\u0131 y\u00fcklenemedi.")
		return
	}
	passwordError := ""
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			a.renderError(w, http.StatusBadRequest, "Form verisi okunamad\u0131.")
			return
		}
		admin := a.currentAdmin(r)
		if admin == nil {
			a.renderError(w, http.StatusUnauthorized, "Y\u00f6netici oturumu bulunamad\u0131.")
			return
		}
		currentPassword := r.FormValue("current_password")
		newPassword := r.FormValue("new_password")
		confirmPassword := r.FormValue("confirm_password")
		switch {
		case currentPassword == "":
			passwordError = "L\u00fctfen mevcut \u015fifrenizi girin."
		case hashPassword(currentPassword) != admin.PasswordHash:
			passwordError = "Mevcut \u015fifreniz hatal\u0131."
		case len(newPassword) < 8:
			passwordError = "Yeni \u015fifreniz en az 8 karakter olmal\u0131d\u0131r."
		case newPassword != confirmPassword:
			passwordError = "Yeni \u015fifre ile do\u011frulama birbiriyle e\u015fle\u015fmiyor."
		case currentPassword == newPassword:
			passwordError = "L\u00fctfen mevcut \u015fifreden farkl\u0131 bir \u015fifre se\u00e7in."
		default:
			newHash := hashPassword(newPassword)
			if err := updateAdminUserPassword(r.Context(), a.db, admin.ID, newHash); err != nil {
				a.renderError(w, http.StatusInternalServerError, "\u015eifre g\u00fcncellenemedi.")
				return
			}
			a.clearSession(w)
			a.setFlash(w, "Y\u00f6netici \u015fifresi g\u00fcncellendi. L\u00fctfen yeniden giri\u015f yap\u0131n.")
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
	}
	layout.Flash = a.readFlash(w, r)
	a.render(w, "templates/admin/password.html", AdminPasswordData{
		AdminLayoutData: layout,
		PasswordError:   passwordError,
	})
}

func (a *App) handleAdminPages(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Sayfalar", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Sayfalar y\u00fcklenemedi.")
		return
	}
	layout.Flash = a.readFlash(w, r)
	pages, err := listPages(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Sayfalar listelenemedi.")
		return
	}
	a.render(w, "templates/admin/pages.html", AdminPagesData{AdminLayoutData: layout, Pages: pages})
}

func (a *App) handleAdminPageEdit(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Sayfa D\u00fczenle", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Sayfa y\u00fcklenemedi.")
		return
	}
	slug := r.URL.Query().Get("slug")
	if slug == "" {
		slug = "about"
	}
	lang := LangTR
	page, err := getPage(r.Context(), a.db, slug, lang)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		a.renderError(w, http.StatusInternalServerError, "Sayfa y\u00fcklenemedi.")
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		page = Page{Slug: slug, Language: lang}
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			a.renderError(w, http.StatusBadRequest, "Form verisi okunamad\u0131.")
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
			a.renderError(w, http.StatusInternalServerError, "Sayfa kaydedilemedi.")
			return
		}
		a.invalidateContentCache()
		a.setFlash(w, "Sayfa g\u00fcncellendi.")
		http.Redirect(w, r, "/admin/pages", http.StatusSeeOther)
		return
	}
	a.render(w, "templates/admin/page_edit.html", AdminPageEditData{AdminLayoutData: layout, Page: page})
}

func (a *App) handleAdminSections(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Ana Sayfa B\u00f6l\u00fcmleri", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "B\u00f6l\u00fcmler y\u00fcklenemedi.")
		return
	}
	layout.Flash = a.readFlash(w, r)
	sections, err := listSections(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "B\u00f6l\u00fcmler listelenemedi.")
		return
	}
	a.render(w, "templates/admin/sections.html", AdminSectionsData{AdminLayoutData: layout, Sections: sections})
}

func (a *App) handleAdminSectionEdit(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "B\u00f6l\u00fcm D\u00fczenle", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "B\u00f6l\u00fcm y\u00fcklenemedi.")
		return
	}
	id := parseID(r.URL.Query().Get("id"))
	section, err := getSection(r.Context(), a.db, id)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "B\u00f6l\u00fcm y\u00fcklenemedi.")
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			a.renderError(w, http.StatusBadRequest, "Form verisi okunamad\u0131.")
			return
		}
		section.Title = r.FormValue("title")
		section.Subtitle = r.FormValue("subtitle")
		section.Body = r.FormValue("body")
		section.CTAName = r.FormValue("cta_name")
		section.CTAURL = r.FormValue("cta_url")
		section.SortOrder = int(parseID(r.FormValue("sort_order")))
		if err := updateSection(r.Context(), a.db, section); err != nil {
			a.renderError(w, http.StatusInternalServerError, "B\u00f6l\u00fcm kaydedilemedi.")
			return
		}
		a.invalidateContentCache()
		a.setFlash(w, "B\u00f6l\u00fcm g\u00fcncellendi.")
		http.Redirect(w, r, "/admin/sections", http.StatusSeeOther)
		return
	}
	a.render(w, "templates/admin/section_edit.html", AdminSectionEditData{AdminLayoutData: layout, Section: section})
}

func (a *App) handleAdminPosts(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Duyurular", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Duyurular y\u00fcklenemedi.")
		return
	}
	layout.Flash = a.readFlash(w, r)
	posts, err := listPosts(r.Context(), a.db, true)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Duyurular listelenemedi.")
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
	layout, err := adminLayoutData(r.Context(), a.db, "Duyuru D\u00fczenle", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Duyuru y\u00fcklenemedi.")
		return
	}
	post := Post{}
	var cover *MediaAsset
	if id > 0 {
		post, err = getPostByID(r.Context(), a.db, id)
		if err != nil {
			a.renderError(w, http.StatusInternalServerError, "Duyuru y\u00fcklenemedi.")
			return
		}
		if post.CoverImageID.Valid {
			cover, err = getMedia(r.Context(), a.db, post.CoverImageID.Int64)
			if err != nil {
				a.renderError(w, http.StatusInternalServerError, "Kapak g\u00f6rseli y\u00fcklenemedi.")
				return
			}
		}
	}
	if r.Method == http.MethodPost {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			a.renderError(w, http.StatusBadRequest, "Form verisi okunamad\u0131.")
			return
		}
		coverImageID := post.CoverImageID
		if r.FormValue("remove_cover") == "1" {
			coverImageID = sqlNullInt64{}
		}
		file, header, fileErr := r.FormFile("cover_upload")
		if fileErr == nil {
			defer file.Close()
			fileName := fileNameForUpload(header.Filename)
			target := filepath.Join(a.config.UploadDir, fileName)
			if err := saveUploadedFile(file, target); err != nil {
				a.renderError(w, http.StatusInternalServerError, "Dosya kaydedilemedi.")
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
				a.renderError(w, http.StatusInternalServerError, "G\u00f6rsel kayd\u0131 olu\u015fturulamad\u0131.")
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
			a.renderError(w, http.StatusInternalServerError, "Duyuru kaydedilemedi.")
			return
		}
		a.invalidateContentCache()
		a.setFlash(w, "Duyuru kaydedildi.")
		http.Redirect(w, r, "/admin/posts", http.StatusSeeOther)
		return
	}
	a.render(w, "templates/admin/post_edit.html", AdminPostEditData{AdminLayoutData: layout, Post: post, Cover: cover})
}

func (a *App) handleAdminPostDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.renderError(w, http.StatusBadRequest, "Ge\u00e7ersiz istek.")
		return
	}
	if err := deletePost(r.Context(), a.db, parseID(r.FormValue("id"))); err != nil {
		a.renderError(w, http.StatusInternalServerError, "Duyuru silinemedi.")
		return
	}
	a.invalidateContentCache()
	a.setFlash(w, "Duyuru silindi.")
	http.Redirect(w, r, "/admin/posts", http.StatusSeeOther)
}

func (a *App) handleAdminPostPublish(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.renderError(w, http.StatusBadRequest, "Ge\u00e7ersiz istek.")
		return
	}
	id := parseID(r.FormValue("id"))
	post, err := getPostByID(r.Context(), a.db, id)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Duyuru y\u00fcklenemedi.")
		return
	}
	post.Published = true
	post.PublishedAt = nullTime(time.Now())
	if _, err := savePost(r.Context(), a.db, post); err != nil {
		a.renderError(w, http.StatusInternalServerError, "Duyuru yay\u0131mlanamad\u0131.")
		return
	}
	a.invalidateContentCache()
	a.setFlash(w, "Duyuru yay\u0131na al\u0131nd\u0131.")
	http.Redirect(w, r, "/admin/posts", http.StatusSeeOther)
}

func (a *App) handleAdminMedia(w http.ResponseWriter, r *http.Request) {
	layout, err := adminLayoutData(r.Context(), a.db, "Medya K\u00fct\u00fcphanesi", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Medya k\u00fct\u00fcphanesi y\u00fcklenemedi.")
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			a.renderError(w, http.StatusBadRequest, "Y\u00fckleme verisi okunamad\u0131.")
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			a.renderError(w, http.StatusBadRequest, "Dosya se\u00e7ilmelidir.")
			return
		}
		defer file.Close()
		fileName := fileNameForUpload(header.Filename)
		target := filepath.Join(a.config.UploadDir, fileName)
		if err := saveUploadedFile(file, target); err != nil {
			a.renderError(w, http.StatusInternalServerError, "Dosya kaydedilemedi.")
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
			a.renderError(w, http.StatusInternalServerError, "G\u00f6rsel kayd\u0131 olu\u015fturulamad\u0131.")
			return
		}
		a.invalidateContentCache()
		a.setFlash(w, "Medya y\u00fcklendi.")
		http.Redirect(w, r, "/admin/media", http.StatusSeeOther)
		return
	}
	layout.Flash = a.readFlash(w, r)
	media, err := listMedia(r.Context(), a.db)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Medya listelenemedi.")
		return
	}
	a.render(w, "templates/admin/media.html", AdminMediaData{AdminLayoutData: layout, Media: media})
}

func (a *App) handleAdminMediaDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.renderError(w, http.StatusBadRequest, "Ge\u00e7ersiz istek.")
		return
	}
	id := parseID(r.FormValue("id"))
	media, err := getMedia(r.Context(), a.db, id)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Medya y\u00fcklenemedi.")
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
	layout, err := adminLayoutData(r.Context(), a.db, "\u0130leti\u015fim Mesajlar\u0131", a.currentAdmin(r))
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Mesajlar y\u00fcklenemedi.")
		return
	}
	contacts, err := listContactSubmissions(r.Context(), a.db, 0)
	if err != nil {
		a.renderError(w, http.StatusInternalServerError, "Mesajlar listelenemedi.")
		return
	}
	a.render(w, "templates/admin/contacts.html", AdminContactsData{AdminLayoutData: layout, Contacts: contacts})
}

func (a *App) handleAdminContactDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.renderError(w, http.StatusBadRequest, "Ge\u00e7ersiz istek.")
		return
	}
	if err := deleteContactSubmission(r.Context(), a.db, parseID(r.FormValue("id"))); err != nil {
		a.renderError(w, http.StatusInternalServerError, "Mesaj silinemedi.")
		return
	}
	a.setFlash(w, "Mesaj silindi.")
	http.Redirect(w, r, "/admin/contacts", http.StatusSeeOther)
}
