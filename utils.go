package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strings"
	"unicode"
)

func hashPassword(password string) string {
	sum := sha256.Sum256([]byte("association-site:" + password))
	return hex.EncodeToString(sum[:])
}

func slugify(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	replacer := strings.NewReplacer(
		"ç", "c",
		"ğ", "g",
		"ı", "i",
		"ö", "o",
		"ş", "s",
		"ü", "u",
	)
	input = replacer.Replace(input)
	var b strings.Builder
	lastDash := false
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "post"
	}
	return regexp.MustCompile(`-+`).ReplaceAllString(out, "-")
}

func nl2br(value string) template.HTML {
	escaped := html.EscapeString(strings.TrimSpace(value))
	escaped = strings.ReplaceAll(escaped, "\n", "<br>")
	return template.HTML(escaped)
}

func valueByLang(lang Language, trValue, enValue string) string {
	if lang == LangEN {
		if strings.TrimSpace(enValue) != "" {
			return enValue
		}
		return trValue
	}
	if strings.TrimSpace(trValue) != "" {
		return trValue
	}
	return enValue
}

func pagePath(lang Language, slug string) string {
	if slug == "home" {
		if lang == LangEN {
			return "/en"
		}
		return "/"
	}
	if lang == LangEN {
		return fmt.Sprintf("/en/%s", slug)
	}
	return "/" + slug
}

func announcementPath(lang Language, slug string) string {
	if lang == LangEN {
		return "/en/announcements/" + slug
	}
	return "/announcements/" + slug
}

func sectionLabel(key string) string {
	switch strings.TrimSpace(strings.ToLower(key)) {
	case "hero":
		return "Karşılama"
	case "about":
		return "Hakkımızda"
	case "activities":
		return "Faaliyetler"
	case "cta":
		return "Çağrı Alanı"
	default:
		return key
	}
}
