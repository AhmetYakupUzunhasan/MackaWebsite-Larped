package main

import "time"

type Language string

const (
	LangTR Language = "tr"
	LangEN Language = "en"
)

type SiteSettings struct {
	ID                int64
	AssociationNameTR string
	AssociationNameEN string
	TaglineTR         string
	TaglineEN         string
	FooterTextTR      string
	FooterTextEN      string
	ContactEmail      string
	ContactPhone      string
	AddressTR         string
	AddressEN         string
	InstagramURL      string
	FacebookURL       string
	LinkedInURL       string
	NavHomeTR         string
	NavHomeEN         string
	NavAboutTR        string
	NavAboutEN        string
	NavContactTR      string
	NavContactEN      string
	NavPostsTR        string
	NavPostsEN        string
	SEODescriptionTR  string
	SEODescriptionEN  string
}

type Page struct {
	ID             int64
	Slug           string
	Language       Language
	Title          string
	Intro          string
	Body           string
	SEODescription string
	UpdatedAt      time.Time
}

type PageSection struct {
	ID         int64
	PageSlug   string
	SectionKey string
	Language   Language
	Title      string
	Subtitle   string
	Body       string
	CTAName    string
	CTAURL     string
	ImageID    sqlNullInt64
	SortOrder  int
	UpdatedAt  time.Time
}

type MediaAsset struct {
	ID           int64
	Title        string
	AltTR        string
	AltEN        string
	FileName     string
	OriginalName string
	MimeType     string
	CreatedAt    time.Time
}

type Post struct {
	ID           int64
	Slug         string
	TitleTR      string
	TitleEN      string
	SummaryTR    string
	SummaryEN    string
	BodyTR       string
	BodyEN       string
	CoverImageID sqlNullInt64
	Published    bool
	PublishedAt  sqlNullTime
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ContactSubmission struct {
	ID        int64
	Name      string
	Email     string
	Subject   string
	Message   string
	Language  Language
	CreatedAt time.Time
}

type AdminUser struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

type HomePageData struct {
	Settings     SiteSettings
	Lang         Language
	Page         Page
	Sections     []SectionWithMedia
	LatestPosts  []PostCard
	FlashSuccess string
}

type SectionWithMedia struct {
	PageSection
	Media *MediaAsset
}

type PostCard struct {
	Post
	Media *MediaAsset
}

type PageData struct {
	Settings     SiteSettings
	Lang         Language
	Page         Page
	FlashSuccess string
}

type PostListData struct {
	Settings SiteSettings
	Lang     Language
	Page     Page
	Posts    []PostCard
}

type PostDetailData struct {
	Settings SiteSettings
	Lang     Language
	Page     Page
	Post     Post
	Media    *MediaAsset
}

type AdminLayoutData struct {
	Title    string
	User     *AdminUser
	Settings SiteSettings
	Flash    string
}

type DashboardData struct {
	AdminLayoutData
	PageCount      int
	SectionCount   int
	PostCount      int
	MediaCount     int
	ContactCount   int
	RecentContacts []ContactSubmission
}

type AdminSettingsData struct {
	AdminLayoutData
	Form SiteSettings
}

type AdminPagesData struct {
	AdminLayoutData
	Pages []Page
}

type AdminPageEditData struct {
	AdminLayoutData
	Page Page
}

type AdminSectionsData struct {
	AdminLayoutData
	Sections []PageSection
	Media    []MediaAsset
}

type AdminSectionEditData struct {
	AdminLayoutData
	Section PageSection
	Media   []MediaAsset
}

type AdminPostsData struct {
	AdminLayoutData
	Posts []Post
}

type AdminPostEditData struct {
	AdminLayoutData
	Post  Post
	Media []MediaAsset
}

type AdminMediaData struct {
	AdminLayoutData
	Media []MediaAsset
}

type AdminContactsData struct {
	AdminLayoutData
	Contacts []ContactSubmission
}
