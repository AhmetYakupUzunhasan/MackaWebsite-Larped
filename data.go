package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"time"
)

func runMigrations(ctx context.Context, db *sql.DB, source embed.FS) error {
	files, err := fs.Glob(source, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, file := range files {
		content, err := source.ReadFile(file)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("migration %s: %w", file, err)
		}
	}
	return nil
}

func seedData(ctx context.Context, db *sql.DB, cfg Config) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO site_settings (
			id, association_name_tr, association_name_en, tagline_tr, tagline_en, footer_text_tr, footer_text_en,
			contact_email, contact_phone, address_tr, address_en, instagram_url, facebook_url, linkedin_url,
			nav_home_tr, nav_home_en, nav_about_tr, nav_about_en, nav_contact_tr, nav_contact_en,
			nav_posts_tr, nav_posts_en, seo_description_tr, seo_description_en
		) VALUES (
			1, 'Dayanışma Derneği', 'Community Solidarity Association',
			'Birlikte daha kapsayıcı bir gelecek için çalışıyoruz.',
			'Working together for a more inclusive future.',
			'Toplumu güçlendiren ortak iyilik girişimleri.',
			'Community-led initiatives that create public good.',
			'hello@example.org', '+90 555 000 00 00',
			'İstanbul, Türkiye', 'Istanbul, Turkiye',
			'', '', '',
			'Ana Sayfa', 'Home', 'Hakkımızda', 'About', 'İletişim', 'Contact', 'Duyurular', 'Announcements',
			'Derneğin faaliyetleri, duyuruları ve iletişim bilgileri.',
			'Association activities, announcements, and contact information.'
		)
		ON CONFLICT(id) DO NOTHING
	`); err != nil {
		return err
	}

	pages := []Page{
		{Slug: "home", Language: LangTR, Title: "Birlikte Üretiyor, Birlikte Güçleniyoruz", Intro: "Derneğimizin amacı, toplumsal dayanışmanın kalıcı ve kapsayıcı yapılarla büyümesine katkıda bulunmak.", Body: "Gönüllüler, destekçiler ve yerel paydaşlarla birlikte üretilen programlar aracılığıyla daha adil bir gelecek için çalışıyoruz.", SEODescription: "Derneğimizin hikâyesi, faaliyetleri ve güncel duyuruları."},
		{Slug: "home", Language: LangEN, Title: "Growing Stronger Through Solidarity", Intro: "Our association builds long-term community support through inclusive programs and partnerships.", Body: "We work with volunteers, supporters, and local partners to create meaningful public benefit.", SEODescription: "Learn about our mission, activities, and latest announcements."},
		{Slug: "about", Language: LangTR, Title: "Hakkımızda", Intro: "Ortak bir amaç etrafında bir araya gelen bir topluluğuz.", Body: "Derneğimiz; eğitim, dayanışma ve toplumsal katılım alanlarında üretilen projelerle etkisini büyütüyor.\n\nKurumsal hafızamızı ve sahadaki öğrenimlerimizi, yeni iş birliklerine açık bir şekilde paylaşıyoruz.", SEODescription: "Derneğimizin vizyonu, değerleri ve çalışma alanı."},
		{Slug: "about", Language: LangEN, Title: "About", Intro: "We are a community brought together around a shared purpose.", Body: "Our association expands its impact through projects focused on education, solidarity, and civic participation.\n\nWe openly share our institutional learning to strengthen new collaborations.", SEODescription: "Vision, values, and programs of the association."},
		{Slug: "contact", Language: LangTR, Title: "İletişim", Intro: "Bizimle iletişime geçmek, iş birliği önermek ya da soru sormak için formu kullanabilirsiniz.", Body: "En kısa sürede size dönüş yapmaya çalışıyoruz.", SEODescription: "Dernek iletişim formu ve iletişim bilgileri."},
		{Slug: "contact", Language: LangEN, Title: "Contact", Intro: "Use the form below to contact us, suggest a collaboration, or ask a question.", Body: "We will get back to you as soon as possible.", SEODescription: "Association contact form and contact details."},
	}
	for _, page := range pages {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO pages (slug, language, title, intro, body, seo_description, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(slug, language) DO NOTHING
		`, page.Slug, page.Language, page.Title, page.Intro, page.Body, page.SEODescription); err != nil {
			return err
		}
	}

	sections := []PageSection{
		{PageSlug: "home", SectionKey: "hero", Language: LangTR, Title: "Toplumsal dayanışma için birlikte hareket ediyoruz", Subtitle: "Yerel ihtiyaçları dinleyen, sürdürülebilir ve insan odaklı programlar.", Body: "Gönüllü ağımız, iş birliklerimiz ve sahadaki deneyimimizle toplumsal faydayı büyüten projeler üretiyoruz.", CTAName: "Bize Ulaşın", CTAURL: "/contact", SortOrder: 1},
		{PageSlug: "home", SectionKey: "hero", Language: LangEN, Title: "We move together for community solidarity", Subtitle: "Sustainable, people-centered programs shaped by local needs.", Body: "Through volunteers, partnerships, and field experience, we create projects that strengthen public benefit.", CTAName: "Contact Us", CTAURL: "/en/contact", SortOrder: 1},
		{PageSlug: "home", SectionKey: "about", Language: LangTR, Title: "Neden varız?", Subtitle: "Kapsayıcı topluluklar için ortak üretim", Body: "Derneğimiz, farklı kesimlerin bir araya gelerek bilgiyi, emeği ve kaynakları paylaştığı bir dayanışmanın alanını kurar.", CTAName: "Hakkımızda", CTAURL: "/about", SortOrder: 2},
		{PageSlug: "home", SectionKey: "about", Language: LangEN, Title: "Why we exist", Subtitle: "Shared work for inclusive communities", Body: "Our association creates space for people to share knowledge, time, and resources in support of collective wellbeing.", CTAName: "About Us", CTAURL: "/en/about", SortOrder: 2},
		{PageSlug: "home", SectionKey: "activities", Language: LangTR, Title: "Neler yapıyoruz?", Subtitle: "Sahadan öğreniyor, birlikte geliştiriyoruz", Body: "Atölyeler, dayanışma programları, bilgilendirme buluşmaları ve yerel ortaklıklar aracılığıyla uzun vadeli etki oluşturuyoruz.", CTAName: "Duyurular", CTAURL: "/announcements", SortOrder: 3},
		{PageSlug: "home", SectionKey: "activities", Language: LangEN, Title: "What we do", Subtitle: "We learn from the field and build together", Body: "We create long-term impact through workshops, support programs, community meetings, and local partnerships.", CTAName: "Announcements", CTAURL: "/en/announcements", SortOrder: 3},
		{PageSlug: "home", SectionKey: "cta", Language: LangTR, Title: "Birlikte çalışalım", Subtitle: "Ortaklık, destek ya da bilgi talepleriniz için bize ulaşın.", Body: "Topluma değer katan yeni iş birliklerine her zaman açığız.", CTAName: "Mesaj Gönder", CTAURL: "/contact", SortOrder: 4},
		{PageSlug: "home", SectionKey: "cta", Language: LangEN, Title: "Let us collaborate", Subtitle: "Reach out for partnerships, support, or information.", Body: "We are always open to new collaborations that create public value.", CTAName: "Send a Message", CTAURL: "/en/contact", SortOrder: 4},
	}
	for _, section := range sections {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO page_sections
				(page_slug, section_key, language, title, subtitle, body, cta_name, cta_url, sort_order, updated_at)
			SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP
			WHERE NOT EXISTS (
				SELECT 1 FROM page_sections WHERE page_slug = ? AND section_key = ? AND language = ?
			)
		`, section.PageSlug, section.SectionKey, section.Language, section.Title, section.Subtitle, section.Body, section.CTAName, section.CTAURL, section.SortOrder, section.PageSlug, section.SectionKey, section.Language); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO admin_users (username, password_hash)
		SELECT ?, ?
		WHERE NOT EXISTS (SELECT 1 FROM admin_users)
	`, cfg.AdminUser, hashPassword(cfg.AdminPass)); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO posts (slug, title_tr, title_en, summary_tr, summary_en, body_tr, body_en, published, published_at, updated_at)
		SELECT
			'ilk-duyuru',
			'İlk duyuru',
			'First announcement',
			'Dernek sitesinin ilk duyurusu yayında.',
			'The first announcement for the association site is live.',
			'Bu alan yönetim panelinden tamamen düzenlenebilir ilk duyuru örneğidir.',
			'This is the first fully editable announcement managed from the admin panel.',
			1,
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP
		WHERE NOT EXISTS (SELECT 1 FROM posts)
	`); err != nil {
		return err
	}

	if err := normalizeLegacyTurkishContent(ctx, tx); err != nil {
		return err
	}

	return tx.Commit()
}

func normalizeLegacyTurkishContent(ctx context.Context, tx *sql.Tx) error {
	updates := []struct {
		query string
		args  []any
	}{
		{
			query: `UPDATE site_settings SET
				association_name_tr = 'Dayanışma Derneği',
				tagline_tr = 'Birlikte daha kapsayıcı bir gelecek için çalışıyoruz.',
				footer_text_tr = 'Toplumu güçlendiren ortak iyilik girişimleri.',
				address_tr = 'İstanbul, Türkiye',
				nav_about_tr = 'Hakkımızda',
				nav_contact_tr = 'İletişim',
				seo_description_tr = 'Derneğin faaliyetleri, duyuruları ve iletişim bilgileri.'
				WHERE id = 1 AND (
					association_name_tr = 'Dayanisma Dernegi' OR
					tagline_tr = 'Birlikte daha kapsayici bir gelecek icin calisiyoruz.' OR
					footer_text_tr = 'Toplumu guclendiren ortak iyilik girisimleri.' OR
					address_tr = 'Istanbul, Turkiye' OR
					nav_about_tr = 'Hakkimizda' OR
					nav_contact_tr = 'Iletisim'
				)`,
		},
		{
			query: `UPDATE pages SET title = ?, intro = ?, body = ?, seo_description = ? WHERE slug = 'home' AND language = 'tr' AND title = 'Birlikte Uretiyor, Birlikte Gucleniyoruz'`,
			args:  []any{"Birlikte Üretiyor, Birlikte Güçleniyoruz", "Derneğimizin amacı, toplumsal dayanışmanın kalıcı ve kapsayıcı yapılarla büyümesine katkıda bulunmak.", "Gönüllüler, destekçiler ve yerel paydaşlarla birlikte üretilen programlar aracılığıyla daha adil bir gelecek için çalışıyoruz.", "Derneğimizin hikâyesi, faaliyetleri ve güncel duyuruları."},
		},
		{
			query: `UPDATE pages SET title = ?, intro = ?, body = ?, seo_description = ? WHERE slug = 'about' AND language = 'tr' AND title = 'Hakkimizda'`,
			args:  []any{"Hakkımızda", "Ortak bir amaç etrafında bir araya gelen bir topluluğuz.", "Derneğimiz; eğitim, dayanışma ve toplumsal katılım alanlarında üretilen projelerle etkisini büyütüyor.\n\nKurumsal hafızamızı ve sahadaki öğrenimlerimizi, yeni iş birliklerine açık bir şekilde paylaşıyoruz.", "Derneğimizin vizyonu, değerleri ve çalışma alanı."},
		},
		{
			query: `UPDATE pages SET title = ?, intro = ?, body = ?, seo_description = ? WHERE slug = 'contact' AND language = 'tr' AND title = 'Iletisim'`,
			args:  []any{"İletişim", "Bizimle iletişime geçmek, iş birliği önermek ya da soru sormak için formu kullanabilirsiniz.", "En kısa sürede size dönüş yapmaya çalışıyoruz.", "Dernek iletişim formu ve iletişim bilgileri."},
		},
		{
			query: `UPDATE page_sections SET title = ?, subtitle = ?, body = ?, cta_name = ? WHERE page_slug = 'home' AND section_key = 'hero' AND language = 'tr'`,
			args:  []any{"Toplumsal dayanışma için birlikte hareket ediyoruz", "Yerel ihtiyaçları dinleyen, sürdürülebilir ve insan odaklı programlar.", "Gönüllü ağımız, iş birliklerimiz ve sahadaki deneyimimizle toplumsal faydayı büyüten projeler üretiyoruz.", "Bize Ulaşın"},
		},
		{
			query: `UPDATE page_sections SET title = ?, subtitle = ?, body = ?, cta_name = ? WHERE page_slug = 'home' AND section_key = 'about' AND language = 'tr'`,
			args:  []any{"Neden varız?", "Kapsayıcı topluluklar için ortak üretim", "Derneğimiz, farklı kesimlerin bir araya gelerek bilgiyi, emeği ve kaynakları paylaştığı bir dayanışmanın alanını kurar.", "Hakkımızda"},
		},
		{
			query: `UPDATE page_sections SET title = ?, subtitle = ?, body = ? WHERE page_slug = 'home' AND section_key = 'activities' AND language = 'tr'`,
			args:  []any{"Neler yapıyoruz?", "Sahadan öğreniyor, birlikte geliştiriyoruz", "Atölyeler, dayanışma programları, bilgilendirme buluşmaları ve yerel ortaklıklar aracılığıyla uzun vadeli etki oluşturuyoruz."},
		},
		{
			query: `UPDATE page_sections SET title = ?, subtitle = ?, body = ?, cta_name = ? WHERE page_slug = 'home' AND section_key = 'cta' AND language = 'tr'`,
			args:  []any{"Birlikte çalışalım", "Ortaklık, destek ya da bilgi talepleriniz için bize ulaşın.", "Topluma değer katan yeni iş birliklerine her zaman açığız.", "Mesaj Gönder"},
		},
		{
			query: `UPDATE posts SET title_tr = ?, summary_tr = ?, body_tr = ? WHERE slug = 'ilk-duyuru' AND title_tr = 'Ilk duyuru'`,
			args:  []any{"İlk duyuru", "Dernek sitesinin ilk duyurusu yayında.", "Bu alan yönetim panelinden tamamen düzenlenebilir ilk duyuru örneğidir."},
		},
	}

	for _, update := range updates {
		if _, err := tx.ExecContext(ctx, update.query, update.args...); err != nil {
			return err
		}
	}
	return nil
}

func getSiteSettings(ctx context.Context, db *sql.DB) (SiteSettings, error) {
	var s SiteSettings
	err := db.QueryRowContext(ctx, `
		SELECT id, association_name_tr, association_name_en, tagline_tr, tagline_en, footer_text_tr, footer_text_en,
			contact_email, contact_phone, address_tr, address_en, instagram_url, facebook_url, linkedin_url,
			nav_home_tr, nav_home_en, nav_about_tr, nav_about_en, nav_contact_tr, nav_contact_en,
			nav_posts_tr, nav_posts_en, seo_description_tr, seo_description_en
		FROM site_settings WHERE id = 1
	`).Scan(
		&s.ID, &s.AssociationNameTR, &s.AssociationNameEN, &s.TaglineTR, &s.TaglineEN, &s.FooterTextTR, &s.FooterTextEN,
		&s.ContactEmail, &s.ContactPhone, &s.AddressTR, &s.AddressEN, &s.InstagramURL, &s.FacebookURL, &s.LinkedInURL,
		&s.NavHomeTR, &s.NavHomeEN, &s.NavAboutTR, &s.NavAboutEN, &s.NavContactTR, &s.NavContactEN,
		&s.NavPostsTR, &s.NavPostsEN, &s.SEODescriptionTR, &s.SEODescriptionEN,
	)
	return s, err
}

func updateSiteSettings(ctx context.Context, db *sql.DB, s SiteSettings) error {
	_, err := db.ExecContext(ctx, `
		UPDATE site_settings SET
			association_name_tr = ?, association_name_en = ?, tagline_tr = ?, tagline_en = ?,
			footer_text_tr = ?, footer_text_en = ?, contact_email = ?, contact_phone = ?,
			address_tr = ?, address_en = ?, instagram_url = ?, facebook_url = ?, linkedin_url = ?,
			nav_home_tr = ?, nav_home_en = ?, nav_about_tr = ?, nav_about_en = ?,
			nav_contact_tr = ?, nav_contact_en = ?, nav_posts_tr = ?, nav_posts_en = ?,
			seo_description_tr = ?, seo_description_en = ?
		WHERE id = 1
	`, s.AssociationNameTR, s.AssociationNameEN, s.TaglineTR, s.TaglineEN, s.FooterTextTR, s.FooterTextEN,
		s.ContactEmail, s.ContactPhone, s.AddressTR, s.AddressEN, s.InstagramURL, s.FacebookURL, s.LinkedInURL,
		s.NavHomeTR, s.NavHomeEN, s.NavAboutTR, s.NavAboutEN, s.NavContactTR, s.NavContactEN, s.NavPostsTR, s.NavPostsEN,
		s.SEODescriptionTR, s.SEODescriptionEN,
	)
	return err
}

func listPages(ctx context.Context, db *sql.DB) ([]Page, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, slug, language, title, intro, body, seo_description, updated_at
		FROM pages WHERE language = 'tr' ORDER BY slug
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []Page
	for rows.Next() {
		var p Page
		if err := rows.Scan(&p.ID, &p.Slug, &p.Language, &p.Title, &p.Intro, &p.Body, &p.SEODescription, &p.UpdatedAt); err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

func getPage(ctx context.Context, db *sql.DB, slug string, lang Language) (Page, error) {
	var p Page
	err := db.QueryRowContext(ctx, `
		SELECT id, slug, language, title, intro, body, seo_description, updated_at
		FROM pages WHERE slug = ? AND language = ?
	`, slug, lang).Scan(&p.ID, &p.Slug, &p.Language, &p.Title, &p.Intro, &p.Body, &p.SEODescription, &p.UpdatedAt)
	return p, err
}

func upsertPage(ctx context.Context, db *sql.DB, p Page) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO pages (slug, language, title, intro, body, seo_description, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(slug, language) DO UPDATE SET
			title = excluded.title,
			intro = excluded.intro,
			body = excluded.body,
			seo_description = excluded.seo_description,
			updated_at = CURRENT_TIMESTAMP
	`, p.Slug, p.Language, p.Title, p.Intro, p.Body, p.SEODescription)
	return err
}

func listSections(ctx context.Context, db *sql.DB) ([]PageSection, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, page_slug, section_key, language, title, subtitle, body, cta_name, cta_url, image_id, sort_order, updated_at
		FROM page_sections
		WHERE language = 'tr'
		ORDER BY page_slug, sort_order
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sections []PageSection
	for rows.Next() {
		var s PageSection
		if err := rows.Scan(&s.ID, &s.PageSlug, &s.SectionKey, &s.Language, &s.Title, &s.Subtitle, &s.Body, &s.CTAName, &s.CTAURL, &s.ImageID, &s.SortOrder, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sections = append(sections, s)
	}
	return sections, rows.Err()
}

func getSection(ctx context.Context, db *sql.DB, id int64) (PageSection, error) {
	var s PageSection
	err := db.QueryRowContext(ctx, `
		SELECT id, page_slug, section_key, language, title, subtitle, body, cta_name, cta_url, image_id, sort_order, updated_at
		FROM page_sections WHERE id = ?
	`, id).Scan(&s.ID, &s.PageSlug, &s.SectionKey, &s.Language, &s.Title, &s.Subtitle, &s.Body, &s.CTAName, &s.CTAURL, &s.ImageID, &s.SortOrder, &s.UpdatedAt)
	return s, err
}

func getSectionsForPage(ctx context.Context, db *sql.DB, pageSlug string, lang Language) ([]SectionWithMedia, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT s.id, s.page_slug, s.section_key, s.language, s.title, s.subtitle, s.body, s.cta_name, s.cta_url, s.image_id, s.sort_order, s.updated_at,
			m.id, m.title, m.alt_tr, m.alt_en, m.file_name, m.original_name, m.mime_type, m.created_at
		FROM page_sections s
		LEFT JOIN media m ON s.image_id = m.id
		WHERE s.page_slug = ? AND s.language = ?
		ORDER BY s.sort_order
	`, pageSlug, lang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sections []SectionWithMedia
	for rows.Next() {
		var item SectionWithMedia
		var mediaID sqlNullInt64
		var mediaTitle, mediaAltTR, mediaAltEN, fileName, originalName, mimeType sqlNullString
		var mediaCreated sqlNullTime
		if err := rows.Scan(
			&item.ID, &item.PageSlug, &item.SectionKey, &item.Language, &item.Title, &item.Subtitle, &item.Body, &item.CTAName, &item.CTAURL, &item.ImageID, &item.SortOrder, &item.UpdatedAt,
			&mediaID, &mediaTitle, &mediaAltTR, &mediaAltEN, &fileName, &originalName, &mimeType, &mediaCreated,
		); err != nil {
			return nil, err
		}
		if mediaID.Valid {
			item.Media = &MediaAsset{
				ID:           mediaID.Int64,
				Title:        mediaTitle.String,
				AltTR:        mediaAltTR.String,
				AltEN:        mediaAltEN.String,
				FileName:     fileName.String,
				OriginalName: originalName.String,
				MimeType:     mimeType.String,
				CreatedAt:    mediaCreated.Time,
			}
		}
		sections = append(sections, item)
	}
	return sections, rows.Err()
}

func updateSection(ctx context.Context, db *sql.DB, s PageSection) error {
	_, err := db.ExecContext(ctx, `
		UPDATE page_sections SET
			title = ?, subtitle = ?, body = ?, cta_name = ?, cta_url = ?, image_id = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, s.Title, s.Subtitle, s.Body, s.CTAName, s.CTAURL, s.ImageID, s.SortOrder, s.ID)
	return err
}

func listMedia(ctx context.Context, db *sql.DB) ([]MediaAsset, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, title, alt_tr, alt_en, file_name, original_name, mime_type, created_at
		FROM media ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var media []MediaAsset
	for rows.Next() {
		var m MediaAsset
		if err := rows.Scan(&m.ID, &m.Title, &m.AltTR, &m.AltEN, &m.FileName, &m.OriginalName, &m.MimeType, &m.CreatedAt); err != nil {
			return nil, err
		}
		media = append(media, m)
	}
	return media, rows.Err()
}

func getMedia(ctx context.Context, db *sql.DB, id int64) (*MediaAsset, error) {
	var m MediaAsset
	err := db.QueryRowContext(ctx, `
		SELECT id, title, alt_tr, alt_en, file_name, original_name, mime_type, created_at
		FROM media WHERE id = ?
	`, id).Scan(&m.ID, &m.Title, &m.AltTR, &m.AltEN, &m.FileName, &m.OriginalName, &m.MimeType, &m.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func insertMedia(ctx context.Context, db *sql.DB, m MediaAsset) (int64, error) {
	res, err := db.ExecContext(ctx, `
		INSERT INTO media (title, alt_tr, alt_en, file_name, original_name, mime_type)
		VALUES (?, ?, ?, ?, ?, ?)
	`, m.Title, m.AltTR, m.AltEN, m.FileName, m.OriginalName, m.MimeType)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func deleteMedia(ctx context.Context, db *sql.DB, id int64) error {
	_, err := db.ExecContext(ctx, `DELETE FROM media WHERE id = ?`, id)
	return err
}

func listPosts(ctx context.Context, db *sql.DB, includeDrafts bool) ([]Post, error) {
	query := `
		SELECT id, slug, title_tr, title_en, summary_tr, summary_en, body_tr, body_en, cover_image_id, published, published_at, created_at, updated_at
		FROM posts
	`
	if !includeDrafts {
		query += ` WHERE published = 1`
	}
	query += ` ORDER BY COALESCE(published_at, created_at) DESC, id DESC`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.Slug, &p.TitleTR, &p.TitleEN, &p.SummaryTR, &p.SummaryEN, &p.BodyTR, &p.BodyEN, &p.CoverImageID, &p.Published, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

func listPostCards(ctx context.Context, db *sql.DB, includeDrafts bool, limit int) ([]PostCard, error) {
	posts, err := listPosts(ctx, db, includeDrafts)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(posts) > limit {
		posts = posts[:limit]
	}
	out := make([]PostCard, 0, len(posts))
	for _, p := range posts {
		var media *MediaAsset
		if p.CoverImageID.Valid {
			media, err = getMedia(ctx, db, p.CoverImageID.Int64)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, PostCard{Post: p, Media: media})
	}
	return out, nil
}

func getPostByID(ctx context.Context, db *sql.DB, id int64) (Post, error) {
	var p Post
	err := db.QueryRowContext(ctx, `
		SELECT id, slug, title_tr, title_en, summary_tr, summary_en, body_tr, body_en, cover_image_id, published, published_at, created_at, updated_at
		FROM posts WHERE id = ?
	`, id).Scan(&p.ID, &p.Slug, &p.TitleTR, &p.TitleEN, &p.SummaryTR, &p.SummaryEN, &p.BodyTR, &p.BodyEN, &p.CoverImageID, &p.Published, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func getPostBySlug(ctx context.Context, db *sql.DB, slug string, includeDrafts bool) (Post, error) {
	query := `
		SELECT id, slug, title_tr, title_en, summary_tr, summary_en, body_tr, body_en, cover_image_id, published, published_at, created_at, updated_at
		FROM posts WHERE slug = ?
	`
	if !includeDrafts {
		query += ` AND published = 1`
	}
	var p Post
	err := db.QueryRowContext(ctx, query, slug).Scan(&p.ID, &p.Slug, &p.TitleTR, &p.TitleEN, &p.SummaryTR, &p.SummaryEN, &p.BodyTR, &p.BodyEN, &p.CoverImageID, &p.Published, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func savePost(ctx context.Context, db *sql.DB, p Post) (int64, error) {
	if p.Published && !p.PublishedAt.Valid {
		p.PublishedAt = nullTime(time.Now())
	}
	if p.ID == 0 {
		res, err := db.ExecContext(ctx, `
			INSERT INTO posts (slug, title_tr, title_en, summary_tr, summary_en, body_tr, body_en, cover_image_id, published, published_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		`, p.Slug, p.TitleTR, p.TitleEN, p.SummaryTR, p.SummaryEN, p.BodyTR, p.BodyEN, p.CoverImageID, p.Published, p.PublishedAt)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	_, err := db.ExecContext(ctx, `
		UPDATE posts SET
			slug = ?, title_tr = ?, title_en = ?, summary_tr = ?, summary_en = ?,
			body_tr = ?, body_en = ?, cover_image_id = ?, published = ?, published_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, p.Slug, p.TitleTR, p.TitleEN, p.SummaryTR, p.SummaryEN, p.BodyTR, p.BodyEN, p.CoverImageID, p.Published, p.PublishedAt, p.ID)
	return p.ID, err
}

func deletePost(ctx context.Context, db *sql.DB, id int64) error {
	_, err := db.ExecContext(ctx, `DELETE FROM posts WHERE id = ?`, id)
	return err
}

func saveContactSubmission(ctx context.Context, db *sql.DB, c ContactSubmission) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO contact_submissions (name, email, subject, message, language)
		VALUES (?, ?, ?, ?, ?)
	`, c.Name, c.Email, c.Subject, c.Message, c.Language)
	return err
}

func listContactSubmissions(ctx context.Context, db *sql.DB, limit int) ([]ContactSubmission, error) {
	query := `
		SELECT id, name, email, subject, message, language, created_at
		FROM contact_submissions ORDER BY created_at DESC, id DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []ContactSubmission
	for rows.Next() {
		var c ContactSubmission
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Subject, &c.Message, &c.Language, &c.CreatedAt); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

func getAdminUserByUsername(ctx context.Context, db *sql.DB, username string) (*AdminUser, error) {
	var user AdminUser
	err := db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, created_at
		FROM admin_users WHERE username = ?
	`, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func getAdminUserByID(ctx context.Context, db *sql.DB, id int64) (*AdminUser, error) {
	var user AdminUser
	err := db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, created_at
		FROM admin_users WHERE id = ?
	`, id).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func getDashboardCounts(ctx context.Context, db *sql.DB) (map[string]int, error) {
	out := map[string]int{}
	queries := map[string]string{
		"pages":    "SELECT COUNT(*) FROM pages",
		"sections": "SELECT COUNT(*) FROM page_sections",
		"posts":    "SELECT COUNT(*) FROM posts",
		"media":    "SELECT COUNT(*) FROM media",
		"contacts": "SELECT COUNT(*) FROM contact_submissions",
	}
	for key, query := range queries {
		var count int
		if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return nil, err
		}
		out[key] = count
	}
	return out, nil
}
