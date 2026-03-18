package main

import (
	"context"
	"sync"
)

type contentCache struct {
	mu             sync.RWMutex
	settings       *SiteSettings
	pages          map[string]Page
	sections       map[string][]SectionWithMedia
	publishedPosts []PostCard
	postDetails    map[string]cachedPostDetail
}

type cachedPostDetail struct {
	post  Post
	media *MediaAsset
}

func newContentCache() *contentCache {
	return &contentCache{
		pages:       map[string]Page{},
		sections:    map[string][]SectionWithMedia{},
		postDetails: map[string]cachedPostDetail{},
	}
}

func (a *App) invalidateContentCache() {
	a.cache.mu.Lock()
	defer a.cache.mu.Unlock()
	a.cache.settings = nil
	a.cache.pages = map[string]Page{}
	a.cache.sections = map[string][]SectionWithMedia{}
	a.cache.publishedPosts = nil
	a.cache.postDetails = map[string]cachedPostDetail{}
}

func (a *App) cachedSettings(ctx context.Context) (SiteSettings, error) {
	a.cache.mu.RLock()
	if a.cache.settings != nil {
		value := *a.cache.settings
		a.cache.mu.RUnlock()
		return value, nil
	}
	a.cache.mu.RUnlock()

	settings, err := getSiteSettings(ctx, a.db)
	if err != nil {
		return SiteSettings{}, err
	}
	a.cache.mu.Lock()
	a.cache.settings = &settings
	a.cache.mu.Unlock()
	return settings, nil
}

func (a *App) cachedPage(ctx context.Context, slug string, lang Language) (Page, error) {
	key := string(lang) + ":" + slug
	a.cache.mu.RLock()
	if value, ok := a.cache.pages[key]; ok {
		a.cache.mu.RUnlock()
		return value, nil
	}
	a.cache.mu.RUnlock()

	page, err := getPage(ctx, a.db, slug, lang)
	if err != nil {
		return Page{}, err
	}
	a.cache.mu.Lock()
	a.cache.pages[key] = page
	a.cache.mu.Unlock()
	return page, nil
}

func (a *App) cachedSections(ctx context.Context, pageSlug string, lang Language) ([]SectionWithMedia, error) {
	key := string(lang) + ":" + pageSlug
	a.cache.mu.RLock()
	if value, ok := a.cache.sections[key]; ok {
		out := append([]SectionWithMedia(nil), value...)
		a.cache.mu.RUnlock()
		return out, nil
	}
	a.cache.mu.RUnlock()

	sections, err := getSectionsForPage(ctx, a.db, pageSlug, lang)
	if err != nil {
		return nil, err
	}
	a.cache.mu.Lock()
	a.cache.sections[key] = append([]SectionWithMedia(nil), sections...)
	a.cache.mu.Unlock()
	return sections, nil
}

func (a *App) cachedPublishedPosts(ctx context.Context, limit int) ([]PostCard, error) {
	a.cache.mu.RLock()
	if a.cache.publishedPosts != nil {
		out := append([]PostCard(nil), a.cache.publishedPosts...)
		a.cache.mu.RUnlock()
		if limit > 0 && len(out) > limit {
			out = out[:limit]
		}
		return out, nil
	}
	a.cache.mu.RUnlock()

	posts, err := listPostCards(ctx, a.db, false, 0)
	if err != nil {
		return nil, err
	}
	a.cache.mu.Lock()
	a.cache.publishedPosts = append([]PostCard(nil), posts...)
	a.cache.mu.Unlock()
	if limit > 0 && len(posts) > limit {
		posts = posts[:limit]
	}
	return posts, nil
}

func (a *App) cachedPostDetail(ctx context.Context, slug string) (Post, *MediaAsset, error) {
	a.cache.mu.RLock()
	if value, ok := a.cache.postDetails[slug]; ok {
		a.cache.mu.RUnlock()
		return value.post, value.media, nil
	}
	a.cache.mu.RUnlock()

	post, err := getPostBySlug(ctx, a.db, slug, false)
	if err != nil {
		return Post{}, nil, err
	}
	var media *MediaAsset
	if post.CoverImageID.Valid {
		media, err = getMedia(ctx, a.db, post.CoverImageID.Int64)
		if err != nil {
			return Post{}, nil, err
		}
	}
	a.cache.mu.Lock()
	a.cache.postDetails[slug] = cachedPostDetail{post: post, media: media}
	a.cache.mu.Unlock()
	return post, media, nil
}
