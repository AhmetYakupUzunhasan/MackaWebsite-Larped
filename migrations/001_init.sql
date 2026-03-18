CREATE TABLE IF NOT EXISTS admin_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS site_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    association_name_tr TEXT NOT NULL DEFAULT '',
    association_name_en TEXT NOT NULL DEFAULT '',
    tagline_tr TEXT NOT NULL DEFAULT '',
    tagline_en TEXT NOT NULL DEFAULT '',
    footer_text_tr TEXT NOT NULL DEFAULT '',
    footer_text_en TEXT NOT NULL DEFAULT '',
    contact_email TEXT NOT NULL DEFAULT '',
    contact_phone TEXT NOT NULL DEFAULT '',
    address_tr TEXT NOT NULL DEFAULT '',
    address_en TEXT NOT NULL DEFAULT '',
    instagram_url TEXT NOT NULL DEFAULT '',
    facebook_url TEXT NOT NULL DEFAULT '',
    linkedin_url TEXT NOT NULL DEFAULT '',
    nav_home_tr TEXT NOT NULL DEFAULT '',
    nav_home_en TEXT NOT NULL DEFAULT '',
    nav_about_tr TEXT NOT NULL DEFAULT '',
    nav_about_en TEXT NOT NULL DEFAULT '',
    nav_contact_tr TEXT NOT NULL DEFAULT '',
    nav_contact_en TEXT NOT NULL DEFAULT '',
    nav_posts_tr TEXT NOT NULL DEFAULT '',
    nav_posts_en TEXT NOT NULL DEFAULT '',
    seo_description_tr TEXT NOT NULL DEFAULT '',
    seo_description_en TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL,
    language TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    intro TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    seo_description TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(slug, language)
);

CREATE TABLE IF NOT EXISTS media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL DEFAULT '',
    alt_tr TEXT NOT NULL DEFAULT '',
    alt_en TEXT NOT NULL DEFAULT '',
    file_name TEXT NOT NULL,
    original_name TEXT NOT NULL DEFAULT '',
    mime_type TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS page_sections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    page_slug TEXT NOT NULL,
    section_key TEXT NOT NULL,
    language TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    subtitle TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    cta_name TEXT NOT NULL DEFAULT '',
    cta_url TEXT NOT NULL DEFAULT '',
    image_id INTEGER,
    sort_order INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(image_id) REFERENCES media(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE,
    title_tr TEXT NOT NULL DEFAULT '',
    title_en TEXT NOT NULL DEFAULT '',
    summary_tr TEXT NOT NULL DEFAULT '',
    summary_en TEXT NOT NULL DEFAULT '',
    body_tr TEXT NOT NULL DEFAULT '',
    body_en TEXT NOT NULL DEFAULT '',
    cover_image_id INTEGER,
    published INTEGER NOT NULL DEFAULT 0,
    published_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(cover_image_id) REFERENCES media(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS contact_submissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    subject TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL,
    language TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
